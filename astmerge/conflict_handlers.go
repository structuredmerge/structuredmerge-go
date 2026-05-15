package astmerge

const (
	GenericIndependentCommutativeInsertionsHandler = "generic-independent-commutative-insertions"
	GenericKeyedMemberEditHandler                  = "generic-keyed-member-edit"
)

func ExecuteGenericConflictHandler(handlerCase GenericConflictHandlerCase) GenericConflictHandlerResult {
	switch handlerCase.HandlerID {
	case GenericIndependentCommutativeInsertionsHandler:
		return executeIndependentCommutativeInsertions(handlerCase)
	case GenericKeyedMemberEditHandler:
		return executeIndependentKeyedMemberEdits(handlerCase)
	default:
		return GenericConflictHandlerResult{
			Resolved:    false,
			Diagnostics: []string{"unsupported generic conflict handler"},
		}
	}
}

func executeIndependentCommutativeInsertions(handlerCase GenericConflictHandlerCase) GenericConflictHandlerResult {
	if handlerCase.ParentPolicy != "commutative" {
		return GenericConflictHandlerResult{
			Resolved:    false,
			Diagnostics: []string{"independent insertion handler requires a commutative parent"},
		}
	}

	seen := map[string]bool{}
	merged := make([]HandlerChildNode, 0, len(handlerCase.BaseChildren)+len(handlerCase.LeftInsertions)+len(handlerCase.RightInsertions))
	appendUnique := func(nodes []HandlerChildNode) {
		for _, node := range nodes {
			key := node.Signature
			if key == "" {
				key = node.NodeID
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			merged = append(merged, node)
		}
	}

	appendUnique(handlerCase.BaseChildren)
	appendUnique(handlerCase.LeftInsertions)
	appendUnique(handlerCase.RightInsertions)

	return GenericConflictHandlerResult{
		Resolved:       true,
		MergedChildren: merged,
		Diagnostics:    []string{"independent insertions into a commutative parent were unioned deterministically"},
	}
}

func executeIndependentKeyedMemberEdits(handlerCase GenericConflictHandlerCase) GenericConflictHandlerResult {
	order := make([]string, 0, len(handlerCase.BaseMembers)+len(handlerCase.LeftEdits)+len(handlerCase.RightEdits))
	values := map[string]string{}
	setMember := func(member HandlerKeyedMember) {
		if _, exists := values[member.Key]; !exists {
			order = append(order, member.Key)
		}
		values[member.Key] = member.Value
	}

	for _, member := range handlerCase.BaseMembers {
		setMember(member)
	}
	for _, member := range handlerCase.LeftEdits {
		setMember(member)
	}
	for _, member := range handlerCase.RightEdits {
		if existing, exists := values[member.Key]; exists && existing != member.Value {
			leftEdited := false
			for _, left := range handlerCase.LeftEdits {
				if left.Key == member.Key {
					leftEdited = true
					break
				}
			}
			if leftEdited {
				return GenericConflictHandlerResult{
					Resolved:    false,
					Diagnostics: []string{"keyed member was edited differently on both sides"},
				}
			}
		}
		setMember(member)
	}

	merged := make([]HandlerKeyedMember, 0, len(order))
	for _, key := range order {
		merged = append(merged, HandlerKeyedMember{Key: key, Value: values[key]})
	}

	return GenericConflictHandlerResult{
		Resolved:      true,
		MergedMembers: merged,
		Diagnostics:   []string{"independent keyed member edits were merged by key"},
	}
}
