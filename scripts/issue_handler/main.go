//nolint:stylecheck
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/joho/godotenv"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

var query struct {
	Viewer struct {
		Login     githubv4.String
		CreatedAt githubv4.DateTime
	}
}

type Issue struct {
	ID           githubv4.ID
	Title        githubv4.String
	ProjectCards struct {
		Nodes []struct {
			ID     githubv4.ID
			Column struct {
				ID   githubv4.ID
				Name githubv4.String
			}
		}
	} `graphql:"projectCards(first: 100)"`
	ProjectItems struct {
		Nodes []struct {
			ID githubv4.ID
		}
	} `graphql:"projectItems(first: 100)"`
}

var existingIssues struct {
	Repository struct {
		Issues struct {
			Nodes    []Issue
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
		} `graphql:"issues(first: 5, after: $issueCursor, orderBy: {field: CREATED_AT, direction: DESC})"`
	} `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
}

var mutateIssue struct {
	UpdateProjectV2ItemFieldValueInput struct {
		ProjectV2Item struct {
			ID githubv4.ID
		} `graphql:"projectV2Item"`
	} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
}

var moveIssue struct {
	AddProjectV2ItemById struct {
		Item struct {
			ID githubv4.ID
		} `graphql:"item"`
	} `graphql:"addProjectV2ItemById(input: $input)"`
}

var removeIssueV2 struct {
	DeleteProjectV2Item struct {
		DeletedItemId githubv4.ID `graphql:"deletedItemId"`
	} `graphql:"deleteProjectV2Item(input: $input)"`
}

var removeIssueV1 struct {
	DeleteProjectCard struct {
		DeletedCardId githubv4.ID `graphql:"deletedCardId"`
	} `graphql:"deleteProjectCard(input: $input)"`
}

func main() {
	_ = godotenv.Load()
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GH_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	client := githubv4.NewClient(httpClient)

	ctx := context.Background()

	err := client.Query(ctx, &query, nil)
	if err != nil {
		log.Fatal().Msgf("query failed: %v", err)
	}
	fmt.Println("    Login:", query.Viewer.Login)
	fmt.Println("CreatedAt:", query.Viewer.CreatedAt)

	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String("filecoin-project"),
		"repositoryName":  githubv4.String("bacalhau"),
		"issueCursor":     (*githubv4.String)(nil), // Null after argument to get first page.
	}

	var issues []Issue

	for {
		err := client.Query(ctx, &existingIssues, variables)
		if err != nil {
			log.Fatal().Msgf("query failed: %v", err)
		}

		// fmt.Printf("Issues: %v+\n", existingIssues.Repository.Issues)

		issues = append(issues, existingIssues.Repository.Issues.Nodes...)

		if !existingIssues.Repository.Issues.PageInfo.HasNextPage {
			break
		}

		variables["issueCursor"] = githubv4.NewString(existingIssues.Repository.Issues.PageInfo.EndCursor)
	}

	v2 := false
	pvtID := githubv4.ID("PVT_kwDOAU_qk84AHJ4X")
	for _, issue := range issues {
		// if issue.ID == "I_kwDOGVTvs85XQNVG" && len(issue.ProjectCards.Nodes) > 0 {
		if len(issue.ProjectCards.Nodes) > 0 {
			u := githubv4.AddProjectV2ItemByIdInput{}
			u.ContentID = issue.ID
			u.ProjectID = pvtID
			err := client.Mutate(ctx, &moveIssue, u, nil)
			if err != nil {
				log.Fatal().Msgf("mutation failed: %v", err)
			}

			updateColumn := githubv4.UpdateProjectV2ItemFieldValueInput{}
			updateColumn.FieldID = githubv4.ID("PVTSSF_lADOAU_qk84AHJ4XzgEH4jw") // Status field (replaced column)
			updateColumn.ProjectID = pvtID
			updateColumn.ItemID = moveIssue.AddProjectV2ItemById.Item.ID

			newColumnID := convertColumnV1toColumnV2(issue.ProjectCards.Nodes[0].Column.Name)
			updateColumn.Value = githubv4.ProjectV2FieldValue{SingleSelectOptionID: newColumnID}
			err = client.Mutate(ctx, &mutateIssue, updateColumn, nil)
			if err != nil {
				log.Fatal().Msgf("mutation failed: %v", err)
			}

			if v2 {
				r := githubv4.DeleteProjectV2ItemInput{}
				r.ItemID = issue.ID
				r.ProjectID = issue.ProjectCards.Nodes[0].Column.ID
				err = client.Mutate(ctx, &removeIssueV2, r, nil)
				if err != nil {
					log.Fatal().Msgf("mutation failed: %v", err)
				}
			} else {
				if len(issue.ProjectCards.Nodes) > 0 {
					r := githubv4.DeleteProjectCardInput{}
					r.CardID = issue.ProjectCards.Nodes[0].ID
					r.ClientMutationID = githubv4.NewString(githubv4.String("bacalhau-issue-bot"))
					err = client.Mutate(ctx, &removeIssueV1, r, nil)
					if err != nil {
						log.Fatal().Msgf("mutation failed: %v", err)
					}
					fmt.Printf("Deleted from Project V1: %+v", removeIssueV1.DeleteProjectCard)
				} else {
					fmt.Printf("no project card found for issue %v - skipping", issue.ID)
				}
			}
		}
	}
}

func convertColumnV1toColumnV2(columnName githubv4.String) *githubv4.String {
	projectV2FieldSSFID := ""
	projectV2FieldSSFValue := ""
	switch strings.ToLower(string(columnName)) {
	case strings.ToLower("Must Have for Next Event"):
		projectV2FieldSSFID = "1e2a6912"
		projectV2FieldSSFValue = "Must Have for Next Event"
	case strings.ToLower("In Progress"):
		projectV2FieldSSFID = "47fc9ee4"
		projectV2FieldSSFValue = "In Progress"
	case strings.ToLower("To Celebrate"):
		projectV2FieldSSFID = "e3b53dda"
		projectV2FieldSSFValue = "To Celebrate"
	case strings.ToLower("Done"):
		projectV2FieldSSFID = "98236657"
		projectV2FieldSSFValue = "Done"
	default:
		// "Triage"
		projectV2FieldSSFID = "f17de28f"
		projectV2FieldSSFValue = "Triage"
	}
	_ = projectV2FieldSSFValue
	return githubv4.NewString(githubv4.String(projectV2FieldSSFID))
	// return githubv4.NewString(githubv4.String(projectV2FieldSSFValue))
}

// Test Alterned Title Issue: I_kwDOGVTvs85XQNVG

// OLD: PRO_kwLOGVTvs84A3Zgl
// NEW: PVT_kwDOAU_qk84AHJ4X

// "id": "I_kwDOGVTvs85XQNVG",
// "title": "TEST ISSUE: IGNORE",
// "projectCards": {
//   "edges": [
//     {
//       "node": {
//         "id": "PRC_lALOGVTvs84A3ZglzgUummE",
//         "column": {
//           "id": "PC_lATOGVTvs84A3ZglzgEk_xs",
//           "name": "Must Haves For Next Event"
//         },
//         "project": {
//           "id": "PRO_kwLOGVTvs84A3Zgl"
//         }
//       }
//     }
//   ]

//   {
//     "id": "I_kwDOGVTvs85XQPuM",
//     "title": "TEST ISSUE: 2",
//     "projectCards": {
//       "edges": []
//     },
//     "projectItems": {
//       "nodes": [
//         {
//           "id": "PVTI_lADOAU_qk84AHJ4XzgDmWEM",
//           "project": {
//             "id": "PVT_kwDOAU_qk84AHJ4X"
//           }
//         }
//       ]
//     }
//   },

//   {
//     "id": "I_kwDOGVTvs85XQc9J",
//     "title": "Test ISSUE: IGNORE 2",
//     "projectCards": {
//       "edges": [
//         {
//           "node": {
//             "id": "PRC_lALOGVTvs84A3ZglzgUunOE",
//             "column": {
//               "id": "PC_lATOGVTvs84A3ZglzgEk_xs",
//               "name": "Must Haves For Next Event"
//             },
//             "project": {
//               "id": "PRO_kwLOGVTvs84A3Zgl"
//             }
//           }
//         }
//       ]
//     },
//     "projectItems": {
//       "nodes": []
//     }
//   }

// OLD: PRO_kwLOGVTvs84A3Zgl
// NEW: PC_lATOGVTvs84A3ZglzgEk_xs
