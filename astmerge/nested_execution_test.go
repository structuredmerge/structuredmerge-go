package astmerge

import "testing"

func nestedOperation(address string, family string) DelegatedChildOperation {
	metadata := map[string]any{}
	if family != "" {
		metadata["family"] = family
	}
	return DelegatedChildOperation{
		OperationID:       "operation:" + address,
		ParentOperationID: "parent:merge",
		RequestedStrategy: "delegate_child_surface",
		LanguageChain:     []string{"markdown", "typescript"},
		Surface: DiscoveredSurface{
			SurfaceKind:       "fenced_code_block",
			EffectiveLanguage: "typescript",
			Address:           address,
			Owner: SurfaceOwnerRef{
				Kind:    SurfaceOwnerOwnedRegion,
				Address: "/code_fence/0",
			},
			ReconstructionStrategy: "portable_write",
			Metadata:               metadata,
		},
	}
}

func TestExecuteNestedMergeOrchestratesStages(t *testing.T) {
	address := "document[0] > fenced_code_block[/code_fence/0]"
	nestedOutputs := []DelegatedChildSurfaceOutput{{
		SurfaceAddress: address,
		Output:         "export const feature = true;\n",
	}}
	calls := make([]string, 0, 3)

	result := ExecuteNestedMerge[string](
		nestedOutputs,
		DelegatedChildOutputResolutionOptions{
			DefaultFamily:   "markdown",
			RequestIDPrefix: "nested_markdown_child",
		},
		NestedMergeExecutionCallbacks[string]{
			MergeParent: func() MergeResult[string] {
				calls = append(calls, "merge")
				output := "merged-parent"
				return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &output, Policies: []PolicyReference{}}
			},
			DiscoverOperations: func(mergedOutput string) NestedMergeDiscoveryResult {
				calls = append(calls, "discover:"+mergedOutput)
				return NestedMergeDiscoveryResult{
					OK:          true,
					Diagnostics: []Diagnostic{},
					Operations:  []DelegatedChildOperation{nestedOperation(address, "typescript")},
				}
			},
			ApplyResolvedOutputs: func(mergedOutput string, operations []DelegatedChildOperation, applyPlan DelegatedChildApplyPlan, appliedChildren []AppliedDelegatedChildOutput) MergeResult[string] {
				calls = append(calls, "apply:"+mergedOutput)
				if len(operations) != 1 || operations[0].OperationID != "operation:"+address {
					t.Fatalf("unexpected operations: %+v", operations)
				}
				if len(applyPlan.Entries) != 1 || applyPlan.Entries[0].Family != "typescript" {
					t.Fatalf("unexpected apply plan: %+v", applyPlan)
				}
				if len(appliedChildren) != 1 || appliedChildren[0].OperationID != "operation:"+address {
					t.Fatalf("unexpected applied children: %+v", appliedChildren)
				}
				output := "final-parent"
				return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &output, Policies: []PolicyReference{}}
			},
		},
	)

	if !result.OK || result.Output == nil || *result.Output != "final-parent" {
		t.Fatalf("unexpected nested merge result: %+v", result)
	}
	if len(calls) != 3 || calls[0] != "merge" || calls[1] != "discover:merged-parent" || calls[2] != "apply:merged-parent" {
		t.Fatalf("unexpected execution order: %+v", calls)
	}
}

func TestExecuteNestedMergeReturnsParentFailureUnchanged(t *testing.T) {
	called := false
	result := ExecuteNestedMerge[string](
		nil,
		DelegatedChildOutputResolutionOptions{DefaultFamily: "markdown", RequestIDPrefix: "nested"},
		NestedMergeExecutionCallbacks[string]{
			MergeParent: func() MergeResult[string] {
				return MergeResult[string]{
					OK: false,
					Diagnostics: []Diagnostic{{
						Severity: SeverityError,
						Category: CategoryParseError,
						Message:  "parent failed",
					}},
					Policies: []PolicyReference{},
				}
			},
			DiscoverOperations: func(string) NestedMergeDiscoveryResult {
				called = true
				return NestedMergeDiscoveryResult{OK: true, Diagnostics: []Diagnostic{}}
			},
			ApplyResolvedOutputs: func(string, []DelegatedChildOperation, DelegatedChildApplyPlan, []AppliedDelegatedChildOutput) MergeResult[string] {
				called = true
				output := "unused"
				return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &output, Policies: []PolicyReference{}}
			},
		},
	)

	if result.OK || called {
		t.Fatalf("unexpected parent failure handling: result=%+v called=%v", result, called)
	}
}

func TestExecuteNestedMergeReturnsDiscoveryFailureAndSkipsApply(t *testing.T) {
	applied := false
	result := ExecuteNestedMerge[string](
		nil,
		DelegatedChildOutputResolutionOptions{DefaultFamily: "markdown", RequestIDPrefix: "nested"},
		NestedMergeExecutionCallbacks[string]{
			MergeParent: func() MergeResult[string] {
				output := "merged-parent"
				return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &output, Policies: []PolicyReference{}}
			},
			DiscoverOperations: func(string) NestedMergeDiscoveryResult {
				return NestedMergeDiscoveryResult{
					OK: false,
					Diagnostics: []Diagnostic{{
						Severity: SeverityError,
						Category: CategoryConfigurationError,
						Message:  "discovery failed",
					}},
				}
			},
			ApplyResolvedOutputs: func(string, []DelegatedChildOperation, DelegatedChildApplyPlan, []AppliedDelegatedChildOutput) MergeResult[string] {
				applied = true
				output := "unused"
				return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &output, Policies: []PolicyReference{}}
			},
		},
	)

	if result.OK || applied {
		t.Fatalf("unexpected discovery failure handling: result=%+v applied=%v", result, applied)
	}
}

func TestExecuteDelegatedChildApplyPlanOrchestratesStages(t *testing.T) {
	address := "document[0] > fenced_code_block[/code_fence/0]"
	output := "final-parent"
	result := ExecuteDelegatedChildApplyPlan[string](
		DelegatedChildApplyPlan{
			Entries: []DelegatedChildApplyPlanEntry{{
				RequestID: "projected_child_group:markdown:fence:typescript",
				Family:    "markdown",
				DelegatedGroup: ProjectedChildReviewGroup{
					DelegatedApplyGroup:         "markdown:fence:typescript",
					ParentOperationID:           "parent:merge",
					ChildOperationID:            "operation:" + address,
					DelegatedRuntimeSurfacePath: address,
					CaseIDs:                     []string{},
					DelegatedCaseIDs:            []string{},
				},
				Decision: ReviewDecision{
					RequestID: "projected_child_group:markdown:fence:typescript",
					Action:    ReviewDecisionApplyDelegatedChildGroup,
				},
			}},
		},
		[]AppliedDelegatedChildOutput{{OperationID: "operation:" + address, Output: "child-output\n"}},
		NestedMergeExecutionCallbacks[string]{
			MergeParent: func() MergeResult[string] {
				merged := "merged-parent"
				return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &merged, Policies: []PolicyReference{}}
			},
			DiscoverOperations: func(string) NestedMergeDiscoveryResult {
				return NestedMergeDiscoveryResult{
					OK:          true,
					Diagnostics: []Diagnostic{},
					Operations:  []DelegatedChildOperation{nestedOperation(address, "")},
				}
			},
			ApplyResolvedOutputs: func(_ string, _ []DelegatedChildOperation, applyPlan DelegatedChildApplyPlan, appliedChildren []AppliedDelegatedChildOutput) MergeResult[string] {
				if len(applyPlan.Entries) != 1 || len(appliedChildren) != 1 {
					t.Fatalf("unexpected apply orchestration inputs: %+v %+v", applyPlan, appliedChildren)
				}
				return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &output, Policies: []PolicyReference{}}
			},
		},
	)

	if !result.OK || result.Output == nil || *result.Output != "final-parent" {
		t.Fatalf("unexpected delegated child apply plan execution result: %+v", result)
	}
}

func TestExecuteReviewedNestedMergeUsesAcceptedReviewState(t *testing.T) {
	address := "document[0] > fenced_code_block[/code_fence/0]"
	output := "final-parent"
	result := ExecuteReviewedNestedMerge[string](
		DelegatedChildGroupReviewState{
			Requests: []ReviewRequest{},
			AcceptedGroups: []ProjectedChildReviewGroup{{
				DelegatedApplyGroup:         "markdown:fence:typescript",
				ParentOperationID:           "parent:merge",
				ChildOperationID:            "operation:" + address,
				DelegatedRuntimeSurfacePath: address,
				CaseIDs:                     []string{},
				DelegatedCaseIDs:            []string{},
			}},
			AppliedDecisions: []ReviewDecision{{
				RequestID: "projected_child_group:markdown:fence:typescript",
				Action:    ReviewDecisionApplyDelegatedChildGroup,
			}},
			Diagnostics: []Diagnostic{},
		},
		"markdown",
		[]AppliedDelegatedChildOutput{{OperationID: "operation:" + address, Output: "child-output\n"}},
		NestedMergeExecutionCallbacks[string]{
			MergeParent: func() MergeResult[string] {
				merged := "merged-parent"
				return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &merged, Policies: []PolicyReference{}}
			},
			DiscoverOperations: func(string) NestedMergeDiscoveryResult {
				return NestedMergeDiscoveryResult{
					OK:          true,
					Diagnostics: []Diagnostic{},
					Operations:  []DelegatedChildOperation{nestedOperation(address, "")},
				}
			},
			ApplyResolvedOutputs: func(_ string, _ []DelegatedChildOperation, applyPlan DelegatedChildApplyPlan, _ []AppliedDelegatedChildOutput) MergeResult[string] {
				if len(applyPlan.Entries) != 1 || applyPlan.Entries[0].RequestID != "projected_child_group:markdown:fence:typescript" {
					t.Fatalf("unexpected reviewed nested merge apply plan: %+v", applyPlan)
				}
				return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &output, Policies: []PolicyReference{}}
			},
		},
	)

	if !result.OK || result.Output == nil || *result.Output != "final-parent" {
		t.Fatalf("unexpected reviewed nested merge result: %+v", result)
	}
}

func TestExecuteReviewedNestedExecutionUsesPayload(t *testing.T) {
	address := "document[0] > fenced_code_block[/code_fence/0]"
	output := "final-parent"
	result := ExecuteReviewedNestedExecution[string](
		ReviewedNestedExecutionFor(
			"markdown",
			DelegatedChildGroupReviewState{
				Requests: []ReviewRequest{},
				AcceptedGroups: []ProjectedChildReviewGroup{{
					DelegatedApplyGroup:         "markdown:fence:typescript",
					ParentOperationID:           "parent:merge",
					ChildOperationID:            "operation:" + address,
					DelegatedRuntimeSurfacePath: address,
					CaseIDs:                     []string{},
					DelegatedCaseIDs:            []string{},
				}},
				AppliedDecisions: []ReviewDecision{{
					RequestID: "projected_child_group:markdown:fence:typescript",
					Action:    ReviewDecisionApplyDelegatedChildGroup,
				}},
				Diagnostics: []Diagnostic{},
			},
			[]AppliedDelegatedChildOutput{{OperationID: "operation:" + address, Output: "child-output\n"}},
		),
		NestedMergeExecutionCallbacks[string]{
			MergeParent: func() MergeResult[string] {
				merged := "merged-parent"
				return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &merged, Policies: []PolicyReference{}}
			},
			DiscoverOperations: func(string) NestedMergeDiscoveryResult {
				return NestedMergeDiscoveryResult{
					OK:          true,
					Diagnostics: []Diagnostic{},
					Operations:  []DelegatedChildOperation{nestedOperation(address, "")},
				}
			},
			ApplyResolvedOutputs: func(_ string, _ []DelegatedChildOperation, applyPlan DelegatedChildApplyPlan, appliedChildren []AppliedDelegatedChildOutput) MergeResult[string] {
				if len(applyPlan.Entries) != 1 || applyPlan.Entries[0].RequestID != "projected_child_group:markdown:fence:typescript" {
					t.Fatalf("unexpected reviewed nested execution apply plan: %+v", applyPlan)
				}
				if len(appliedChildren) != 1 || appliedChildren[0].OperationID != "operation:"+address {
					t.Fatalf("unexpected reviewed nested execution applied children: %+v", appliedChildren)
				}
				return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &output, Policies: []PolicyReference{}}
			},
		},
	)

	if !result.OK || result.Output == nil || *result.Output != "final-parent" {
		t.Fatalf("unexpected reviewed nested execution result: %+v", result)
	}
}
