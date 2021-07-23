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

package main

import (
	"bufio"
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"cloud.google.com/go/storage"
	e2e_testing "github.com/GoogleCloudPlatform/opentelemetry-operations-e2e-testing"
	"github.com/alexflint/go-arg"
	"google.golang.org/api/cloudbuild/v1"
)

const (
	passIcon    = ":white_check_mark:"
	skipIcon    = ":leftwards_arrow_with_hook:"
	templateTxt = `<table>
	<thead>
		<tr>
			<th>Repo Name</th>
			<th>Platform</th>
			{{- range $.Scenarios }}
				<th>{{.}}</th>
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
					<td>{{ index (index (index $.RepoToPlatformToScenario $repoName) $platform) $scenario }}</td>
				{{- end }}
			</tr>
		{{- end }}
	{{- end }}
	</tbody>
</table>
`
)

type Args struct {
	e2e_testing.CmdWithProjectId
}

type triggerInfo struct {
	RepoName  string
	Platform  string
	TriggerId string
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
		log.Fatal(err)
	}

	listTriggersRes, err := cloudbuildService.Projects.Triggers.List(args.ProjectID).
		Context(ctx).
		Do()
	if err != nil {
		panic(err)
	}
	repoToTriggerInfo := buildRepoToTriggers(listTriggersRes)

	repoToPlatformToScenario := map[string]map[string]map[string]string{}
	repoNameSet := map[string]struct{}{}
	scenarioSet := map[string]struct{}{}
	platformSet := map[string]struct{}{}
	for repoName, triggerInfos := range repoToTriggerInfo {
		repoNameSet[repoName] = struct{}{}
		platformToScenario := map[string]map[string]string{}
		repoToPlatformToScenario[repoName] = platformToScenario
		for _, triggerInfo := range triggerInfos {
			if triggerInfo.Platform == "build" {
				continue
			}
			scenarioToStatus := map[string]string{}
			platformToScenario[triggerInfo.Platform] = scenarioToStatus
			platformSet[triggerInfo.Platform] = struct{}{}

			passes, skips := getNewestSupportForTrigger(
				ctx,
				args.ProjectID,
				triggerInfo.TriggerId,
				cloudbuildService,
				storageClient,
			)
			for _, pass := range passes {
				scenarioSet[pass] = struct{}{}
				scenarioToStatus[pass] = passIcon
			}
			for _, skip := range skips {
				scenarioSet[skip] = struct{}{}
				scenarioToStatus[skip] = skipIcon
			}
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
		RepoToPlatformToScenario map[string]map[string]map[string]string
	}{repoNames, scenarios, platforms, repoToPlatformToScenario})

	if err != nil {
		panic(err)
	}
}

// buildRepoToTriggers returns a map of repository name onto a map of string
// (platform) to trigger Id
func buildRepoToTriggers(res *cloudbuild.ListBuildTriggersResponse) map[string][]triggerInfo {
	out := map[string][]triggerInfo{}
	for _, trigger := range res.Triggers {
		if !triggerNameRe.MatchString(trigger.Name) {
			continue
		}

		repoName := trigger.Github.Name
		// check tags
		platform := trigger.Tags[1]
		out[repoName] = append(
			out[repoName],
			triggerInfo{RepoName: repoName, Platform: platform, TriggerId: trigger.Id},
		)
	}
	return out
}

// getNewestSupportForTrigger returns the passed and skipped scenarios for the
// given trigger Id.
func getNewestSupportForTrigger(
	ctx context.Context,
	projectId string,
	triggerId string,
	cloudbuildService *cloudbuild.Service,
	storageClient *storage.Client,
) ([]string, []string) {
	listRes, err := cloudbuildService.Projects.Builds.List(projectId).
		Context(ctx).
		Filter(fmt.Sprintf(`trigger_id="%v" AND status="SUCCESS"`, triggerId)).
		PageSize(1).
		Do()
	if err != nil {
		panic(err)
	}
	build := listRes.Builds[0]
	reader, err := storageClient.Bucket(strings.TrimPrefix(build.LogsBucket, "gs://")).
		Object(fmt.Sprintf("log-%v.txt", build.Id)).
		NewReader(ctx)
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	passes := []string{}
	skips := []string{}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if passMatches := scenarioPassRe.FindStringSubmatch(line); passMatches != nil {
			passes = append(passes, passMatches[1])
			continue
		}
		if skipMatches := scenarioSkipRe.FindStringSubmatch(line); skipMatches != nil {
			skips = append(skips, skipMatches[1])
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return passes, skips
}

func sortStringSet(set map[string]struct{}) []string {
	out := []string{}
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
