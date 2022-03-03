package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type ApprovalStatus string

type ApprovalStatusValuesType struct {
	Approved ApprovalStatus
	Rejected ApprovalStatus
	Pending  ApprovalStatus
}

var ApprovalStatusValues = ApprovalStatusValuesType{
	Approved: "Approved",
	Rejected: "Rejected",
	Pending:  "Pending",
}

type ApprovalDecision string

type ApprovalDecisionValuesType struct {
	Approve ApprovalDecision
	Reject  ApprovalDecision
}

var ApprovalDecisionValues = ApprovalDecisionValuesType{
	Approve: "approve",
	Reject:  "reject",
}

type ApprovalDecisionRecord struct {
	Approver string           `json:"approver"`
	Decision ApprovalDecision `json:"decision"`
}

type ApprovalRequest struct {
	Id        int                      `json:"id,omitempty"`
	Requester string                   `json:"requester"`
	Subject   string                   `json:"subject"`
	Status    ApprovalStatus           `json:"status,omitempty"`
	Decisions []ApprovalDecisionRecord `json:"decisions,omitempty"`
}

type Client struct {
	Hostname string
}

func (c *Client) Get(id int) (*ApprovalRequest, error) {
	var arResponse *ApprovalRequest

	resp, err := http.Get(fmt.Sprintf("%s/approval_requests/%d", c.Hostname, id))
	if err != nil {
		return nil, fmt.Errorf("error getting approval request id '%d': %w", id, err)
	}

	err = json.NewDecoder(resp.Body).Decode(arResponse)
	if err != nil {
		return nil, fmt.Errorf("error decoding approval request id '%d': %w", id, err)
	}

	return arResponse, nil

}

func (c *Client) Create(requester string, subject string) (*ApprovalRequest, error) {

	ar := ApprovalRequest{
		Requester: requester,
		Subject: subject,
	}
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(ar)

	resp, err := http.Post(fmt.Sprintf("%s/approval_requests", c.Hostname), "application/json", b)
	if err != nil {
		return nil, fmt.Errorf("error posting approval request: %w", err)
	}

	// repurpose ar to receive the response
	err = json.NewDecoder(resp.Body).Decode(&ar)
	if err != nil {
		return nil, fmt.Errorf("error decoding approval request response: %w", err)
	}

	return &ar, nil

}