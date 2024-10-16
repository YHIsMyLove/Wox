package calculator

import (
	"context"
	"strings"
	"wox/plugin"
	"wox/share"
	"wox/util"
	"wox/util/clipboard"
)

var calculatorIcon = plugin.PluginCalculatorIcon

func init() {
	plugin.AllSystemPlugin = append(plugin.AllSystemPlugin, &CalculatorPlugin{})
}

type CalculatorHistory struct {
	Expression string
	Result     string
	AddDate    string
}

type CalculatorPlugin struct {
	api       plugin.API
	histories []CalculatorHistory
}

func (c *CalculatorPlugin) GetMetadata() plugin.Metadata {
	return plugin.Metadata{
		Id:            "bd723c38-f28d-4152-8621-76fd21d6456e",
		Name:          "Calculator",
		Author:        "Wox Launcher",
		Website:       "https://github.com/Wox-launcher/Wox",
		Version:       "1.0.0",
		MinWoxVersion: "2.0.0",
		Runtime:       "Go",
		Description:   "Calculator for Wox",
		Icon:          calculatorIcon.String(),
		Entry:         "",
		TriggerKeywords: []string{
			"*",
			"calculator",
		},
		Commands: []plugin.MetadataCommand{},
		SupportedOS: []string{
			"Windows",
			"Macos",
			"Linux",
		},
	}
}

func (c *CalculatorPlugin) Init(ctx context.Context, initParams plugin.InitParams) {
	c.api = initParams.API
}

func (c *CalculatorPlugin) Query(ctx context.Context, query plugin.Query) []plugin.QueryResult {
	var results []plugin.QueryResult

	if query.TriggerKeyword == "" {
		//only calculate if query has operators
		if !strings.ContainsAny(query.Search, "+-*/(") {
			return []plugin.QueryResult{}
		}

		val, err := Calculate(query.Search)
		if err != nil {
			return []plugin.QueryResult{}
		}
		result := val.String()

		results = append(results, plugin.QueryResult{
			Title: result,
			Icon:  calculatorIcon,
			Actions: []plugin.QueryResultAction{
				{
					Name: "Copy result",
					Action: func(ctx context.Context, actionContext plugin.ActionContext) {
						c.histories = append(c.histories, CalculatorHistory{
							Expression: query.Search,
							Result:     result,
							AddDate:    util.FormatDateTime(util.GetSystemTime()),
						})
						clipboard.WriteText(result)
					},
				},
			},
		})
	}

	// only show history if query has trigger keyword
	if query.TriggerKeyword != "" {
		val, err := Calculate(query.Search)
		if err == nil {
			result := val.String()
			results = append(results, plugin.QueryResult{
				Title: result,
				Icon:  calculatorIcon,
				Actions: []plugin.QueryResultAction{
					{
						Action: func(ctx context.Context, actionContext plugin.ActionContext) {
							c.histories = append(c.histories, CalculatorHistory{
								Expression: query.Search,
								Result:     result,
								AddDate:    util.FormatDateTime(util.GetSystemTime()),
							})
							clipboard.WriteText(result)
						},
					},
				},
			})
		}

		//show top 500 histories order by desc
		var count = 0
		for i := len(c.histories) - 1; i >= 0; i-- {
			h := c.histories[i]

			count++
			if count >= 500 {
				break
			}

			if strings.Contains(h.Expression, query.Search) || strings.Contains(h.Result, query.Search) {
				results = append(results, plugin.QueryResult{
					Title:    h.Expression,
					SubTitle: h.Result,
					Icon:     calculatorIcon,
					Actions: []plugin.QueryResultAction{
						{
							Name:      "Copy result",
							IsDefault: true,
							Action: func(ctx context.Context, actionContext plugin.ActionContext) {
								clipboard.WriteText(h.Result)
							},
						},
						{
							Name: "Recalculate",
							Action: func(ctx context.Context, actionContext plugin.ActionContext) {
								c.api.ChangeQuery(ctx, share.PlainQuery{
									QueryType: plugin.QueryTypeInput,
									QueryText: h.Expression,
								})
							},
						},
					},
				})
			}
		}

		if len(results) == 0 {
			results = append(results, plugin.QueryResult{
				Title:   "Input expression to calculate",
				Icon:    calculatorIcon,
				Actions: []plugin.QueryResultAction{},
			})
		}
	}

	return results
}
