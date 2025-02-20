// Copyright 2020 Security Scorecard Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package checks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/ossf/scorecard/v4/checker"
	"github.com/ossf/scorecard/v4/clients"
	mockrepo "github.com/ossf/scorecard/v4/clients/mockclients"
	scut "github.com/ossf/scorecard/v4/utests"
)

func TestSAST(t *testing.T) {
	t.Parallel()
	//nolint
	tests := []struct {
		name          string
		prs           []clients.PullRequest
		err           error
		searchresult  clients.SearchResponse
		checkRuns     []clients.CheckRun
		searchRequest clients.SearchRequest
		expected      checker.CheckResult
	}{
		{
			name:         "SAST checker should return failed status when no PRs are found",
			prs:          []clients.PullRequest{},
			searchresult: clients.SearchResponse{},
			checkRuns:    []clients.CheckRun{},
		},
		{
			name:         "SAST checker should return failed status when no PRs are found",
			err:          errors.New("error"),
			prs:          []clients.PullRequest{},
			searchresult: clients.SearchResponse{},
			checkRuns:    []clients.CheckRun{},
			expected:     checker.CheckResult{Score: -1, Pass: false},
		},
		{
			name: "Successful SAST checker should return success status",
			prs: []clients.PullRequest{
				{
					MergedAt: time.Now().Add(time.Hour - 1),
				},
			},
			searchresult: clients.SearchResponse{},
			checkRuns: []clients.CheckRun{
				{
					Status:     "completed",
					Conclusion: "success",
					App: clients.CheckRunApp{
						Slug: "lgtm-com",
					},
				},
			},
			expected: checker.CheckResult{
				Score: 10,
				Pass:  true,
			},
		},
		{
			name: "Failed SAST checker should return success status",
			prs: []clients.PullRequest{
				{
					MergedAt: time.Now().Add(time.Hour - 1),
				},
				{
					MergedAt: time.Now().Add(time.Hour - 10),
				},
				{
					MergedAt: time.Now().Add(time.Hour - 20),
				},
				{
					MergedAt: time.Now().Add(time.Hour - 30),
				},
			},
			searchresult: clients.SearchResponse{Hits: 1, Results: []clients.SearchResult{{
				Path: "test.go",
			}}},
			checkRuns: []clients.CheckRun{
				{
					Status: "completed",
					App: clients.CheckRunApp{
						Slug: "lgtm-com",
					},
				},
			},
			expected: checker.CheckResult{
				Score: 7,
				Pass:  false,
			},
		},
		{
			name: "Failed SAST checker with checkRuns not completed",
			prs: []clients.PullRequest{
				{
					MergedAt: time.Now().Add(time.Hour - 1),
				},
				{
					MergedAt: time.Now().Add(time.Hour - 10),
				},
				{
					MergedAt: time.Now().Add(time.Hour - 20),
				},
				{
					MergedAt: time.Now().Add(time.Hour - 30),
				},
			},
			searchresult: clients.SearchResponse{},
			checkRuns: []clients.CheckRun{
				{
					App: clients.CheckRunApp{
						Slug: "lgtm-com",
					},
				},
			},
			expected: checker.CheckResult{
				Score: 0,
				Pass:  false,
			},
		},
		{
			name: "Failed SAST with PullRequest not merged",
			prs: []clients.PullRequest{
				{
					Number: 1,
				},
			},
			searchresult: clients.SearchResponse{},
			checkRuns: []clients.CheckRun{
				{
					App: clients.CheckRunApp{
						Slug: "lgtm-com",
					},
				},
			},
			expected: checker.CheckResult{
				Score: 0,
				Pass:  false,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		searchRequest := clients.SearchRequest{
			Query: "github/codeql-action/analyze",
			Path:  "/.github/workflows",
		}
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			mockRepo := mockrepo.NewMockRepoClient(ctrl)

			mockRepo.EXPECT().ListMergedPRs().DoAndReturn(func() ([]clients.PullRequest, error) {
				if tt.err != nil {
					return nil, tt.err
				}
				return tt.prs, tt.err
			})
			mockRepo.EXPECT().ListCheckRunsForRef("").Return(tt.checkRuns, nil).AnyTimes()
			mockRepo.EXPECT().Search(searchRequest).Return(tt.searchresult, nil).AnyTimes()

			dl := scut.TestDetailLogger{}
			req := checker.CheckRequest{
				RepoClient: mockRepo,
				Ctx:        context.TODO(),
				Dlogger:    &dl,
			}
			res := SAST(&req)

			if res.Score != tt.expected.Score {
				t.Errorf("Expected score %d, got %d for %v", tt.expected.Score, res.Score, tt.name)
			}
			if res.Pass != tt.expected.Pass {
				t.Errorf("Expected pass %t, got %t for %v", tt.expected.Pass, res.Pass, tt.name)
			}
			ctrl.Finish()
		})
	}
}
