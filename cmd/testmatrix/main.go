// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This script generates the matrix.md file from most recent build of each
// trigger:
//
// ```bash
//	go run cmd/testmatrix/main.go > matrix.md
// ```

package main

import (
	"bufio"
	"context"
	"fmt"
	"html/template"
	"os"
	"regexp"
	"sort"
	"strings"

	"cloud.google.com/go/storage"
	e2e_testing "github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing"
	"github.com/alexflint/go-arg"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/cloudbuild/v1"
)

const (
	pass        status = ":white_check_mark:"
	skip        status = ":leftwards_arrow_with_hook:"
	templateTxt        = `# Matrix of supported scenarios in each ops repo

<table>
	<thead>
		<tr>
			<th>Repo Name</th>
			<th>Platform</th>
			{{- range $.Scenarios }}
				<th>{{ . }}</th>
			{{- end }}
		</tr>
	</thead>
	<tbody>
	{{- range $repoName := $.RepoNames }}
		{{- range $i, $platform := $.Platforms }}
			<tr>
				{{- if eq $i 0 }}
				<td rowspan={{ len $.Platforms }}>
					<a href="https://github.com/GoogleCloudPlatform/{{ $repoName }}">{{ $repoName }}</a>
				</td>
				{{- end }}
				<td>{{ $platform }}</td>
				{{- range $scenario := $.Scenarios }}
					<td>{{ index $.RepoToPlatformToScenario $repoName $platform $scenario }}</td>
				{{- end }}
			</tr>
		{{- end }}
	{{- end }}
	</tbody>
</table>

- *{{ .Pass }} means passing*
- *{{ .Skip }} means not implemented (skipped)*
`
)

type Args struct {
	e2e_testing.CmdWithProjectId
}

type status string

type result struct {
	RepoName string
	Platform string
	Statuses map[string]status
}

var (
	triggerNameRe  = regexp.MustCompile(`^ops-\w+-e2e-.*$`)
	scenarioPassRe = regexp.MustCompile(`: --- PASS:\s+([\w_]+)`)
	scenarioSkipRe = regexp.MustCompile(`: --- SKIP:\s+([\w_]+)`)
)

func main() {
	args := Args{}
	arg.MustParse(&args)

	ctx := context.Background()
	cloudbuildService, err := cloudbuild.NewService(ctx)
	if err != nil {
		panic(err)
	}
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		panic(err)
	}

	// Don't bother going over pages, just use a large page size and look at the
	// first page
	listTriggersRes, err := cloudbuildService.Projects.Triggers.List(args.ProjectID).
		Context(ctx).
		PageSize(128).
		Do()
	if err != nil {
		panic(err)
	}

	g, egCtx := errgroup.WithContext(ctx)
	results := make([]*result, len(listTriggersRes.Triggers))
	for i, trigger := range listTriggersRes.Triggers {
		i := i
		trigger := trigger
		g.Go(func() error {
			res, err := handleTrigger(egCtx, args.ProjectID, trigger, cloudbuildService, storageClient)
			if err != nil {
				return err
			}
			results[i] = res
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		panic(err)
	}

	repoToPlatformToScenario := map[string]map[string]map[string]status{}
	repoNameSet := map[string]struct{}{}
	scenarioSet := map[string]struct{}{}
	platformSet := map[string]struct{}{}
	for _, result := range results {
		if result == nil || result.Platform == "build" {
			continue
		}

		repoNameSet[result.RepoName] = struct{}{}
		if repoToPlatformToScenario[result.RepoName] == nil {
			repoToPlatformToScenario[result.RepoName] = map[string]map[string]status{}
		}
		platformToScenario := repoToPlatformToScenario[result.RepoName]

		platformToScenario[result.Platform] = result.Statuses
		platformSet[result.Platform] = struct{}{}

		for scenario := range result.Statuses {
			scenarioSet[scenario] = struct{}{}
		}
	}
	repoNames := sortStringSet(repoNameSet)
	scenarios := sortStringSet(scenarioSet)
	platforms := sortStringSet(platformSet)

	template := template.Must(template.New("table").Parse(templateTxt))
	err = template.Execute(os.Stdout, struct {
		RepoNames                []string
		Scenarios                []string
		Platforms                []string
		RepoToPlatformToScenario map[string]map[string]map[string]status
		Pass                     status
		Skip                     status
	}{repoNames, scenarios, platforms, repoToPlatformToScenario, pass, skip})

	if err != nil {
		panic(err)
	}
}

// handleTrigger returns the latest results for the given trigger by querying
// builds and logs.
func handleTrigger(
	ctx context.Context,
	projectId string,
	trigger *cloudbuild.BuildTrigger,
	cloudbuildService *cloudbuild.Service,
	storageClient *storage.Client,
) (*result, error) {
	if !triggerNameRe.MatchString(trigger.Name) {
		fmt.Printf("Skipping trigger %v which doesn't match regex", trigger.Name)
		return nil, nil
	}
	res := &result{
		RepoName: trigger.Github.Name,
		Platform: trigger.Tags[1],
		Statuses: make(map[string]status),
	}

	// fetch the latest successful build
	listRes, err := cloudbuildService.Projects.Builds.List(projectId).
		Context(ctx).
		Filter(fmt.Sprintf(`trigger_id="%v" AND status="SUCCESS"`, trigger.Id)).
		PageSize(1).
		Do()
	if err != nil {
		return nil, err
	}

	build := listRes.Builds[0]
	reader, err := storageClient.Bucket(strings.TrimPrefix(build.LogsBucket, "gs://")).
		Object(fmt.Sprintf("log-%v.txt", build.Id)).
		NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if passMatches := scenarioPassRe.FindStringSubmatch(line); passMatches != nil {
			res.Statuses[passMatches[1]] = pass
		} else if skipMatches := scenarioSkipRe.FindStringSubmatch(line); skipMatches != nil {
			res.Statuses[skipMatches[1]] = skip
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return res, nil
}

func sortStringSet(set map[string]struct{}) []string {
	out := []string{}
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
