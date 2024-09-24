package conditionalaccesspolicy

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-azure-sdk/microsoft-graph/common-types/stable"
	"github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/odata"
)

// Copyright (c) HashiCorp Inc. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type ListConditionalAccessPoliciesOperationResponse struct {
	HttpResponse *http.Response
	OData        *odata.OData
	Model        *[]stable.ConditionalAccessPolicy
}

type ListConditionalAccessPoliciesCompleteResult struct {
	LatestHttpResponse *http.Response
	Items              []stable.ConditionalAccessPolicy
}

type ListConditionalAccessPoliciesOperationOptions struct {
	Count     *bool
	Expand    *odata.Expand
	Filter    *string
	Metadata  *odata.Metadata
	OrderBy   *odata.OrderBy
	RetryFunc client.RequestRetryFunc
	Search    *string
	Select    *[]string
	Skip      *int64
	Top       *int64
}

func DefaultListConditionalAccessPoliciesOperationOptions() ListConditionalAccessPoliciesOperationOptions {
	return ListConditionalAccessPoliciesOperationOptions{}
}

func (o ListConditionalAccessPoliciesOperationOptions) ToHeaders() *client.Headers {
	out := client.Headers{}

	return &out
}

func (o ListConditionalAccessPoliciesOperationOptions) ToOData() *odata.Query {
	out := odata.Query{}
	if o.Count != nil {
		out.Count = *o.Count
	}
	if o.Expand != nil {
		out.Expand = *o.Expand
	}
	if o.Filter != nil {
		out.Filter = *o.Filter
	}
	if o.Metadata != nil {
		out.Metadata = *o.Metadata
	}
	if o.OrderBy != nil {
		out.OrderBy = *o.OrderBy
	}
	if o.Search != nil {
		out.Search = *o.Search
	}
	if o.Select != nil {
		out.Select = *o.Select
	}
	if o.Skip != nil {
		out.Skip = int(*o.Skip)
	}
	if o.Top != nil {
		out.Top = int(*o.Top)
	}
	return &out
}

func (o ListConditionalAccessPoliciesOperationOptions) ToQuery() *client.QueryParams {
	out := client.QueryParams{}

	return &out
}

type ListConditionalAccessPoliciesCustomPager struct {
	NextLink *odata.Link `json:"@odata.nextLink"`
}

func (p *ListConditionalAccessPoliciesCustomPager) NextPageLink() *odata.Link {
	defer func() {
		p.NextLink = nil
	}()

	return p.NextLink
}

// ListConditionalAccessPolicies - List policies. Retrieve a list of conditionalAccessPolicy objects.
func (c ConditionalAccessPolicyClient) ListConditionalAccessPolicies(ctx context.Context, options ListConditionalAccessPoliciesOperationOptions) (result ListConditionalAccessPoliciesOperationResponse, err error) {
	opts := client.RequestOptions{
		ContentType: "application/json; charset=utf-8",
		ExpectedStatusCodes: []int{
			http.StatusOK,
		},
		HttpMethod:    http.MethodGet,
		OptionsObject: options,
		Pager:         &ListConditionalAccessPoliciesCustomPager{},
		Path:          "/identity/conditionalAccess/policies",
		RetryFunc:     options.RetryFunc,
	}

	req, err := c.Client.NewRequest(ctx, opts)
	if err != nil {
		return
	}

	var resp *client.Response
	resp, err = req.ExecutePaged(ctx)
	if resp != nil {
		result.OData = resp.OData
		result.HttpResponse = resp.Response
	}
	if err != nil {
		return
	}

	var values struct {
		Values *[]stable.ConditionalAccessPolicy `json:"value"`
	}
	if err = resp.Unmarshal(&values); err != nil {
		return
	}

	result.Model = values.Values

	return
}

// ListConditionalAccessPoliciesComplete retrieves all the results into a single object
func (c ConditionalAccessPolicyClient) ListConditionalAccessPoliciesComplete(ctx context.Context, options ListConditionalAccessPoliciesOperationOptions) (ListConditionalAccessPoliciesCompleteResult, error) {
	return c.ListConditionalAccessPoliciesCompleteMatchingPredicate(ctx, options, ConditionalAccessPolicyOperationPredicate{})
}

// ListConditionalAccessPoliciesCompleteMatchingPredicate retrieves all the results and then applies the predicate
func (c ConditionalAccessPolicyClient) ListConditionalAccessPoliciesCompleteMatchingPredicate(ctx context.Context, options ListConditionalAccessPoliciesOperationOptions, predicate ConditionalAccessPolicyOperationPredicate) (result ListConditionalAccessPoliciesCompleteResult, err error) {
	items := make([]stable.ConditionalAccessPolicy, 0)

	resp, err := c.ListConditionalAccessPolicies(ctx, options)
	if err != nil {
		result.LatestHttpResponse = resp.HttpResponse
		err = fmt.Errorf("loading results: %+v", err)
		return
	}
	if resp.Model != nil {
		for _, v := range *resp.Model {
			if predicate.Matches(v) {
				items = append(items, v)
			}
		}
	}

	result = ListConditionalAccessPoliciesCompleteResult{
		LatestHttpResponse: resp.HttpResponse,
		Items:              items,
	}
	return
}