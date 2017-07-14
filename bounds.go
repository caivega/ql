package ql

// filters represents a set of filter conditions.
// The filter being build is always at index 0.
// Completed filters (once the subtree is walked) are pushed to index > 0
// NOTE: it is considered invalid to have a nil filters object.
type filters struct {
	// conditionals index by column name
	colConditions map[string][]bound
	// expression nodes where column info could not be extracted
	remaining []*binaryOperation
}

func (fs *filters) build(expr expression) {
	switch n := expr.(type) {
	case nil: //no conditions
		return
	case *binaryOperation:
		fs.buildBinOp(n)
	default:
		panic("filters.build(): unknown expression node type")
	}
}

// merge applies all the conditionals specified in src to itself.
func (fs *filters) merge(src *filters) {
	for col, bounds := range src.colConditions {
		if _, colExists := fs.colConditions[col]; colExists {
			for _, b := range bounds {
				fs.colConditions[col] = append(fs.colConditions[col], b)
			}
		} else {
			fs.colConditions[col] = bounds
		}
	}
}

func flipOp(op int) int {
	switch op {
	case '>':
		return '<'
	case le:
		return ge
	case '<':
		return '>'
	case ge:
		return le
	case eq, neq, andand, oror:
		return op
	default:
		panic("flipOp: unknown op type: " + (&binaryOperation{op: op}).String())
	}
}

func (fs *filters) buildBinOp(expr *binaryOperation) {
	// first handle logical operations
	switch expr.op {
	case oror:
		fs.addRemaining(expr)
		return
	case andand:
		fs.build(expr.l)
		fs.build(expr.r)
		return
	}

	if ok, newFilters := fs.tryExtractBinOp(expr.l, expr.r, expr.op); ok {
		fs.merge(newFilters)
	} else if ok, newFilters := fs.tryExtractBinOp(expr.r, expr.l, flipOp(expr.op)); ok {
		fs.merge(newFilters)
	} else {
		//we were unable to extract column information, fallback to resolving the
		//full AST at runtime
		fs.addRemaining(expr)
	}
}

func (fs *filters) addRemaining(expr *binaryOperation) {
	fs.remaining = append(fs.remaining, expr)
}

// generates filters from a relative operation.
// To prevent duplication, we assume column is always LHS.
// caller should call twice to test each position.
func (fs *filters) tryExtractBinOp(l, r expression, op int) (bool, *filters) {
	leftIsColumn, columnName := isColumnExpression(l)
	if !leftIsColumn {
		return false, nil
	}

	switch op {
	case '<':
		return true, makeSingleFilter(columnName, l, r, false, false)
	case le:
		return true, makeSingleFilter(columnName, l, r, true, false)
	case '>':
		return true, makeSingleFilter(columnName, r, l, false, false)
	case ge:
		return true, makeSingleFilter(columnName, r, l, true, false)
	case eq:
		return true, makeSingleFilter(columnName, r, r, true, false)
	case neq:
		return true, makeSingleFilter(columnName, r, r, true, true)
	default:
		panic("filters.tryExtractBinOp: Unhandled op type")
	}
}

func makeSingleFilter(columnName string, min, max expression, inclusive, negated bool) *filters {
	return &filters{
		colConditions: map[string][]bound{
			columnName: []bound{
				bound{
					min: min, max: max, inclusive: inclusive, negated: negated,
				},
			},
		},
	}
}

// TODO: Refactor bounds to be an interface, condense Unary+Binary ops into one interface
// bound represents a constraint which has a minimum and maximum, resolvable at runtime.
type bound struct {
	min       expression
	max       expression
	inclusive bool //set for all bounds checks where val=min or val=max should be included
	negated   bool //logical NOT of the result of the expression.
}

func boundsFromExpr(expr expression) []*filters {
	// one special case - handle a logical OR root
	if o, ok := expr.(*binaryOperation); ok && o.op == oror {
		var out []*filters
		for _, f := range boundsFromExpr(o.l) {
			out = append(out, f)
		}
		for _, f := range boundsFromExpr(o.r) {
			out = append(out, f)
		}
		return out
	}

	f := &filters{colConditions: map[string][]bound{}}
	f.build(expr)
	return []*filters{f}
}
