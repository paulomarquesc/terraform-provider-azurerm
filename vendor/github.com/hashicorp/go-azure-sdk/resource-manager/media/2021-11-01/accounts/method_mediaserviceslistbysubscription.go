package accounts

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/odata"
)

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type MediaservicesListBySubscriptionOperationResponse struct {
	HttpResponse *http.Response
	OData        *odata.OData
	Model        *[]MediaService
}

type MediaservicesListBySubscriptionCompleteResult struct {
	LatestHttpResponse *http.Response
	Items              []MediaService
}

type MediaservicesListBySubscriptionCustomPager struct {
	NextLink *odata.Link `json:"@odata.nextLink"`
}

func (p *MediaservicesListBySubscriptionCustomPager) NextPageLink() *odata.Link {
	defer func() {
		p.NextLink = nil
	}()

	return p.NextLink
}

// MediaservicesListBySubscription ...
func (c AccountsClient) MediaservicesListBySubscription(ctx context.Context, id commonids.SubscriptionId) (result MediaservicesListBySubscriptionOperationResponse, err error) {
	opts := client.RequestOptions{
		ContentType: "application/json; charset=utf-8",
		ExpectedStatusCodes: []int{
			http.StatusOK,
		},
		HttpMethod: http.MethodGet,
		Pager:      &MediaservicesListBySubscriptionCustomPager{},
		Path:       fmt.Sprintf("%s/providers/Microsoft.Media/mediaServices", id.ID()),
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
		Values *[]MediaService `json:"value"`
	}
	if err = resp.Unmarshal(&values); err != nil {
		return
	}

	result.Model = values.Values

	return
}

// MediaservicesListBySubscriptionComplete retrieves all the results into a single object
func (c AccountsClient) MediaservicesListBySubscriptionComplete(ctx context.Context, id commonids.SubscriptionId) (MediaservicesListBySubscriptionCompleteResult, error) {
	return c.MediaservicesListBySubscriptionCompleteMatchingPredicate(ctx, id, MediaServiceOperationPredicate{})
}

// MediaservicesListBySubscriptionCompleteMatchingPredicate retrieves all the results and then applies the predicate
func (c AccountsClient) MediaservicesListBySubscriptionCompleteMatchingPredicate(ctx context.Context, id commonids.SubscriptionId, predicate MediaServiceOperationPredicate) (result MediaservicesListBySubscriptionCompleteResult, err error) {
	items := make([]MediaService, 0)

	resp, err := c.MediaservicesListBySubscription(ctx, id)
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

	result = MediaservicesListBySubscriptionCompleteResult{
		LatestHttpResponse: resp.HttpResponse,
		Items:              items,
	}
	return
}
