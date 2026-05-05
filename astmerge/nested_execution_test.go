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

func TestExecuteReviewedNestedExecutionsPreservesOrder(t *testing.T) {
	markdownAddress := "document[0] > fenced_code_block[/code_fence/0]"
	rubyAddress := "document[0] > ruby_doc_comment[Greeter] > yard_example[1]"
	runs := ExecuteReviewedNestedExecutions(
		[]ReviewedNestedExecution{
			ReviewedNestedExecutionFor(
				"markdown",
				DelegatedChildGroupReviewState{
					Requests: []ReviewRequest{},
					AcceptedGroups: []ProjectedChildReviewGroup{{
						DelegatedApplyGroup:         "nested_markdown_child:0",
						ParentOperationID:           "markdown-document-0",
						ChildOperationID:            "markdown-fence-0",
						DelegatedRuntimeSurfacePath: markdownAddress,
						CaseIDs:                     []string{},
						DelegatedCaseIDs:            []string{},
					}},
					AppliedDecisions: []ReviewDecision{{
						RequestID: "projected_child_group:nested_markdown_child:0",
						Action:    ReviewDecisionApplyDelegatedChildGroup,
					}},
					Diagnostics: []Diagnostic{},
				},
				[]AppliedDelegatedChildOutput{{OperationID: "markdown-fence-0", Output: "child-output\n"}},
			),
			ReviewedNestedExecutionFor(
				"ruby",
				DelegatedChildGroupReviewState{
					Requests: []ReviewRequest{},
					AcceptedGroups: []ProjectedChildReviewGroup{{
						DelegatedApplyGroup:         "nested_ruby_child:0",
						ParentOperationID:           "ruby-doc-comment-0",
						ChildOperationID:            "yard-example-0",
						DelegatedRuntimeSurfacePath: rubyAddress,
						CaseIDs:                     []string{},
						DelegatedCaseIDs:            []string{},
					}},
					AppliedDecisions: []ReviewDecision{{
						RequestID: "projected_child_group:nested_ruby_child:0",
						Action:    ReviewDecisionApplyDelegatedChildGroup,
					}},
					Diagnostics: []Diagnostic{},
				},
				[]AppliedDelegatedChildOutput{{OperationID: "yard-example-0", Output: "Greeter.new.wave\n"}},
			),
		},
		func(execution ReviewedNestedExecution, _ int) NestedMergeExecutionCallbacks[string] {
			return NestedMergeExecutionCallbacks[string]{
				MergeParent: func() MergeResult[string] {
					output := execution.Family + "-merged"
					return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &output, Policies: []PolicyReference{}}
				},
				DiscoverOperations: func(string) NestedMergeDiscoveryResult {
					switch execution.Family {
					case "markdown":
						return NestedMergeDiscoveryResult{
							OK:          true,
							Diagnostics: []Diagnostic{},
							Operations:  []DelegatedChildOperation{nestedOperation(markdownAddress, "typescript")},
						}
					default:
						return NestedMergeDiscoveryResult{
							OK:          true,
							Diagnostics: []Diagnostic{},
							Operations: []DelegatedChildOperation{{
								OperationID:       "yard-example-0",
								ParentOperationID: "ruby-doc-comment-0",
								RequestedStrategy: "delegate_child_surface",
								LanguageChain:     []string{"ruby", "ruby"},
								Surface: DiscoveredSurface{
									SurfaceKind:            "yard_example",
									EffectiveLanguage:      "ruby",
									Address:                rubyAddress,
									Owner:                  SurfaceOwnerRef{Kind: SurfaceOwnerOwnedRegion, Address: "/yard_example/1"},
									ReconstructionStrategy: "portable_write",
									Metadata:               map[string]any{"family": "ruby"},
								},
							}},
						}
					}
				},
				ApplyResolvedOutputs: func(_ string, _ []DelegatedChildOperation, _ DelegatedChildApplyPlan, appliedChildren []AppliedDelegatedChildOutput) MergeResult[string] {
					if len(appliedChildren) != len(execution.AppliedChildren) {
						t.Fatalf("unexpected applied children: %+v", appliedChildren)
					}
					output := execution.Family + "-final"
					return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &output, Policies: []PolicyReference{}}
				},
			}
		},
	)

	if len(runs) != 2 || runs[0].Execution.Family != "markdown" || runs[1].Execution.Family != "ruby" {
		t.Fatalf("unexpected reviewed nested execution order: %+v", runs)
	}
	if runs[0].Result.Output == nil || *runs[0].Result.Output != "markdown-final" || runs[1].Result.Output == nil || *runs[1].Result.Output != "ruby-final" {
		t.Fatalf("unexpected reviewed nested execution results: %+v", runs)
	}
}

func TestExecuteReviewReplayBundleReviewedNestedExecutionsUsesBundle(t *testing.T) {
	runs := ExecuteReviewReplayBundleReviewedNestedExecutions(
		ReviewReplayBundle{
			ReplayContext: ReviewReplayContext{
				Surface:                 "conformance_manifest",
				Families:                []string{"text"},
				RequireExplicitContexts: true,
			},
			Decisions: []ReviewDecision{{
				RequestID: "family_context:text",
				Action:    ReviewDecisionAcceptDefaultContext,
			}},
			ReviewedNestedExecutions: []ReviewedNestedExecution{
				ReviewedNestedExecutionFor(
					"markdown",
					DelegatedChildGroupReviewState{
						Requests: []ReviewRequest{},
						AcceptedGroups: []ProjectedChildReviewGroup{{
							DelegatedApplyGroup:         "nested_markdown_child:0",
							ParentOperationID:           "markdown-document-0",
							ChildOperationID:            "markdown-fence-0",
							DelegatedRuntimeSurfacePath: "document[0] > fenced_code_block[/code_fence/0]",
							CaseIDs:                     []string{},
							DelegatedCaseIDs:            []string{},
						}},
						AppliedDecisions: []ReviewDecision{{
							RequestID: "projected_child_group:nested_markdown_child:0",
							Action:    ReviewDecisionApplyDelegatedChildGroup,
						}},
						Diagnostics: []Diagnostic{},
					},
					[]AppliedDelegatedChildOutput{{OperationID: "markdown-fence-0", Output: "child-output\n"}},
				),
			},
		},
		func(_ ReviewedNestedExecution, _ int) NestedMergeExecutionCallbacks[string] {
			output := "final-parent"
			return NestedMergeExecutionCallbacks[string]{
				MergeParent: func() MergeResult[string] {
					merged := "merged-parent"
					return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &merged, Policies: []PolicyReference{}}
				},
				DiscoverOperations: func(string) NestedMergeDiscoveryResult {
					return NestedMergeDiscoveryResult{OK: true, Diagnostics: []Diagnostic{}, Operations: []DelegatedChildOperation{nestedOperation("document[0] > fenced_code_block[/code_fence/0]", "typescript")}}
				},
				ApplyResolvedOutputs: func(string, []DelegatedChildOperation, DelegatedChildApplyPlan, []AppliedDelegatedChildOutput) MergeResult[string] {
					return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &output, Policies: []PolicyReference{}}
				},
			}
		},
	)

	if len(runs) != 1 || runs[0].Result.Output == nil || *runs[0].Result.Output != "final-parent" {
		t.Fatalf("unexpected replay bundle reviewed nested execution results: %+v", runs)
	}
}

func TestExecuteReviewStateReviewedNestedExecutionsUsesState(t *testing.T) {
	runs := ExecuteReviewStateReviewedNestedExecutions(
		ConformanceManifestReviewState{
			Report: NamedConformanceSuiteReportEnvelope{
				Entries: []NamedConformanceSuiteReport{},
				Summary: ConformanceSuiteSummary{},
			},
			Diagnostics:      []Diagnostic{},
			Requests:         []ReviewRequest{},
			AppliedDecisions: []ReviewDecision{},
			HostHints: ReviewHostHints{
				Interactive:             false,
				RequireExplicitContexts: false,
			},
			ReplayContext: ReviewReplayContext{
				Surface:                 "conformance_manifest",
				Families:                []string{},
				RequireExplicitContexts: false,
			},
			ReviewedNestedExecutions: []ReviewedNestedExecution{
				ReviewedNestedExecutionFor(
					"markdown",
					DelegatedChildGroupReviewState{
						Requests: []ReviewRequest{},
						AcceptedGroups: []ProjectedChildReviewGroup{{
							DelegatedApplyGroup:         "nested_markdown_child:0",
							ParentOperationID:           "markdown-document-0",
							ChildOperationID:            "markdown-fence-0",
							DelegatedRuntimeSurfacePath: "document[0] > fenced_code_block[/code_fence/0]",
							CaseIDs:                     []string{},
							DelegatedCaseIDs:            []string{},
						}},
						AppliedDecisions: []ReviewDecision{{
							RequestID: "projected_child_group:nested_markdown_child:0",
							Action:    ReviewDecisionApplyDelegatedChildGroup,
						}},
						Diagnostics: []Diagnostic{},
					},
					[]AppliedDelegatedChildOutput{{OperationID: "markdown-fence-0", Output: "child-output\n"}},
				),
			},
		},
		func(_ ReviewedNestedExecution, _ int) NestedMergeExecutionCallbacks[string] {
			output := "final-parent"
			return NestedMergeExecutionCallbacks[string]{
				MergeParent: func() MergeResult[string] {
					merged := "merged-parent"
					return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &merged, Policies: []PolicyReference{}}
				},
				DiscoverOperations: func(string) NestedMergeDiscoveryResult {
					return NestedMergeDiscoveryResult{OK: true, Diagnostics: []Diagnostic{}, Operations: []DelegatedChildOperation{nestedOperation("document[0] > fenced_code_block[/code_fence/0]", "typescript")}}
				},
				ApplyResolvedOutputs: func(string, []DelegatedChildOperation, DelegatedChildApplyPlan, []AppliedDelegatedChildOutput) MergeResult[string] {
					return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &output, Policies: []PolicyReference{}}
				},
			}
		},
	)

	if len(runs) != 1 || runs[0].Result.Output == nil || *runs[0].Result.Output != "final-parent" {
		t.Fatalf("unexpected review state reviewed nested execution results: %+v", runs)
	}
}
