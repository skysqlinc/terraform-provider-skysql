package autonomous

import (
	"encoding/json"
	"time"
)

const AutoScaleNodesHorizontalActionGroup = "autoScaleNodesHorizontal"
const AutoScaleNodesVerticalActionGroup = "autoScaleNodesVertical"
const AutoScaleDiskActionGroup = "autoScaleDisk"

// AutoScaleNodesVerticalActionParams is an autoscale action that scales nodes vertically
type AutoScaleNodesVerticalActionParams struct {
	MaxNodeSize string `json:"max_node_size"`
	MinNodeSize string `json:"min_node_size"`
}

// AutoScaleNodesHorizontalActionParams is an autoscale action that scales nodes horizontally
type AutoScaleNodesHorizontalActionParams struct {
	MinNodes int64 `json:"min_nodes"`
	MaxNodes int64 `json:"max_nodes"`
}

// AutoScaleDiskActionParams is an autoscale action that scales disk size
type AutoScaleDiskActionParams struct {
	MaxStorageSizeGBs int64 `json:"max_storage_size_gbs"`
}

type SetAutonomousActionsRequest struct {
	ServiceID   string `json:"service_id"`
	ServiceName string `json:"service_name"`
	Actions     []AutoScaleAction
}

type AutoScaleAction struct {
	Group   string          `json:"group"`
	Enabled bool            `json:"enabled"`
	Params  json.RawMessage `json:"params"`
}

func NewAutoScaleDiskAction(maxStorageSizeGBs int64) AutoScaleAction {
	action := AutoScaleAction{
		Group:   AutoScaleDiskActionGroup,
		Enabled: true,
	}
	rawMessage, err := json.Marshal(&AutoScaleDiskActionParams{
		MaxStorageSizeGBs: maxStorageSizeGBs,
	})

	if err != nil {
		panic(err)
	}

	action.Params = rawMessage

	return action
}

func NewAutoScaleNodesHorizontalAction(minNodes int64, maxNodes int64) AutoScaleAction {
	action := AutoScaleAction{
		Group:   AutoScaleNodesHorizontalActionGroup,
		Enabled: true,
	}
	rawMessage, err := json.Marshal(&AutoScaleNodesHorizontalActionParams{
		MinNodes: minNodes,
		MaxNodes: maxNodes,
	})

	if err != nil {
		panic(err)
	}

	action.Params = rawMessage

	return action
}

func NewAutoScaleNodesVerticalAction(minNodeSize string, maxNodeSize string) AutoScaleAction {
	action := AutoScaleAction{
		Group:   AutoScaleNodesVerticalActionGroup,
		Enabled: true,
	}
	rawMessage, err := json.Marshal(&AutoScaleNodesVerticalActionParams{
		MaxNodeSize: minNodeSize,
		MinNodeSize: maxNodeSize,
	})

	if err != nil {
		panic(err)
	}

	action.Params = rawMessage

	return action
}

type ActionResponse struct {
	Group       string          `json:"group"`
	Enabled     bool            `json:"enabled"`
	Params      json.RawMessage `json:"params"`
	ID          string          `json:"id,omitempty"`
	TenantID    string          `json:"tenant_id"`
	ServiceID   string          `json:"service_id"`
	ServiceName string          `json:"service_name"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
