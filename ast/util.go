package ast

// IsReadOnly checks whether the input ast is readOnly.
func IsReadOnly(node Node) bool {
	switch st := node.(type) {
	case *SelectStmt:
		if st == nil {
			return false
		}

	case *SetOprStmt:
		if st == nil {
			return false
		}
	case *ValuesStmt:
		if st == nil {
			return false
		}
	case *TableStmt:
		if st == nil {
			return false
		}
	case *ExplainStmt:
		// ANALYZE can execute the wrapped statement; classify it conservatively.
		return st != nil && IsReadOnly(st.Stmt)
	default:
		return false
	}
	checker := readOnlyChecker{readOnly: true}
	_, ok := node.Accept(&checker)
	return ok && checker.readOnly
}

type readOnlyChecker struct {
	readOnly bool
}

// Enter implements Visitor interface.
func (checker *readOnlyChecker) Enter(in Node) (out Node, skipChildren bool) {
	switch node := in.(type) {
	case *SelectStmt:
		if node.IntoTable != nil || node.LockTp != SelectLockNone || node.Locking != nil {
			checker.readOnly = false
			return in, true
		}
	case *SetOprStmt:
		if node.Locking != nil {
			checker.readOnly = false
			return in, true
		}
	case *VariableExpr:
		// like func rewriteVariable(), this stands for SetVar.
		if !node.IsSystem && node.Value != nil {
			checker.readOnly = false
			return in, true
		}
	case *InsertStmt, *UpdateStmt, *DeleteStmt, *MergeStmt:
		// Data-modifying statements can appear inside a WITH clause.
		checker.readOnly = false
		return in, true
	}
	return in, false
}

// Leave implements Visitor interface.
func (checker *readOnlyChecker) Leave(in Node) (out Node, ok bool) {
	return in, checker.readOnly
}
