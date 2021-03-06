package taolang

import (
	"math"
	"reflect"
)

// Expression is the interface that is implemented by all expressions.
type Expression interface {
	Evaluate(ctx *Context) Value
}

// Assigner is implemented by those who can be assigned.
type Assigner interface {
	Assign(ctx *Context, value Value)
}

// UnaryExpression is a unary expression.
type UnaryExpression struct {
	op   TokenType
	expr Expression
}

// NewUnaryExpression news a UnaryExpression.
func NewUnaryExpression(op TokenType, expr Expression) *UnaryExpression {
	return &UnaryExpression{
		op:   op,
		expr: expr,
	}
}

// Evaluate implements Expression.
func (u *UnaryExpression) Evaluate(ctx *Context) Value {
	value := u.expr.Evaluate(ctx)
	switch u.op {
	case ttAddition:
		if value.Type != vtNumber {
			panic(NewTypeError("+value is invalid"))
		}
		return ValueFromNumber(+value.number())
	case ttSubtraction:
		if value.Type != vtNumber {
			panic(NewTypeError("-value is invalid"))
		}
		return ValueFromNumber(-value.number())
	case ttLogicalNot:
		return ValueFromBoolean(!value.Truth(ctx))
	case ttBitXor:
		if value.Type != vtNumber {
			panic(NewTypeError("^value is invalid"))
		}
		return ValueFromNumber(^value.number())
	}
	panic(NewSyntaxError("unknown unary operator: %v", u.op))
}

// IncrementDecrementExpression is an a++ / a-- / ++a / --a expressions.
type IncrementDecrementExpression struct {
	prefix bool
	op     TokenType
	expr   Expression
}

// NewIncrementDecrementExpression news an IncrementDecrementExpression.
func NewIncrementDecrementExpression(op TokenType, prefix bool, expr Expression) *IncrementDecrementExpression {
	return &IncrementDecrementExpression{
		prefix: prefix,
		op:     op,
		expr:   expr,
	}
}

// Evaluate implements Expression.
func (i *IncrementDecrementExpression) Evaluate(ctx *Context) Value {
	oldval := i.expr.Evaluate(ctx)
	if oldval.isNumber() {
		assigner, ok := i.expr.(Assigner)
		if !ok {
			panic(NewNotAssignableError(oldval))
		}
		newnum := 0
		switch i.op {
		case ttIncrement:
			newnum = oldval.number() + 1
		case ttDecrement:
			newnum = oldval.number() - 1
		default:
			panic("won't go here")
		}
		newval := ValueFromNumber(newnum)
		assigner.Assign(ctx, newval)
		if i.prefix {
			return newval
		}
		return oldval
	}
	panic(NewNotAssignableError(oldval))
}

// BinaryExpression is a binary expression.
type BinaryExpression struct {
	left  Expression
	op    TokenType
	right Expression
}

// NewBinaryExpression news a BinaryExpression.
func NewBinaryExpression(left Expression, op TokenType, right Expression) *BinaryExpression {
	return &BinaryExpression{
		left:  left,
		op:    op,
		right: right,
	}
}

// Evaluate implements Expression.
func (b *BinaryExpression) Evaluate(ctx *Context) Value {
	op := b.op
	lv, rv := Value{}, Value{}

	// Logical values are evaluated "short-circuit"-ly.
	if op != ttLogicalAnd && op != ttLogicalOr {
		lv = b.left.Evaluate(ctx)
		rv = b.right.Evaluate(ctx)
	}

	lt, rt := lv.Type, rv.Type

	if lt == vtNil && rt == vtNil {
		if op == ttEqual {
			return ValueFromBoolean(true)
		} else if op == ttNotEqual {
			return ValueFromBoolean(false)
		}
	}

	if lt == vtBoolean && rt == vtBoolean {
		switch op {
		case ttEqual:
			return ValueFromBoolean(lv.boolean() == rv.boolean())
		case ttNotEqual:
			return ValueFromBoolean(lv.boolean() != rv.boolean())
		}
	}

	if lt == vtNumber && rt == vtNumber {
		switch op {
		case ttAddition:
			return ValueFromNumber(lv.number() + rv.number())
		case ttSubtraction:
			return ValueFromNumber(lv.number() - rv.number())
		case ttMultiply:
			return ValueFromNumber(lv.number() * rv.number())
		case ttDivision:
			if rv.number() == 0 {
				// TODO
				panic(NewTypeError("divide by zero"))
			}
			return ValueFromNumber(lv.number() / rv.number())
		case ttGreaterThan:
			return ValueFromBoolean(lv.number() > rv.number())
		case ttGreaterThanOrEqual:
			return ValueFromBoolean(lv.number() >= rv.number())
		case ttLessThan:
			return ValueFromBoolean(lv.number() < rv.number())
		case ttLessThanOrEqual:
			return ValueFromBoolean(lv.number() <= rv.number())
		case ttEqual:
			return ValueFromBoolean(lv.number() == rv.number())
		case ttNotEqual:
			return ValueFromBoolean(lv.number() != rv.number())
		case ttPercent:
			return ValueFromNumber(lv.number() % rv.number())
		case ttStarStar:
			// TODO precision lost
			val := math.Pow(float64(lv.number()), float64(rv.number()))
			return ValueFromNumber(int(val))
		case ttLeftShift:
			return ValueFromNumber(lv.number() << uint(rv.number()))
		case ttRightShift:
			return ValueFromNumber(lv.number() >> uint(rv.number()))
		case ttBitAnd:
			return ValueFromNumber(lv.number() & rv.number())
		case ttBitOr:
			return ValueFromNumber(lv.number() | rv.number())
		case ttBitXor:
			return ValueFromNumber(lv.number() ^ rv.number())
		case ttBitAndNot:
			return ValueFromNumber(lv.number() &^ rv.number())
		}
	}

	if lt == vtString && rt == vtString {
		switch op {
		case ttAddition:
			return ValueFromString(lv.str().s + rv.str().s)
		case ttEqual:
			return ValueFromBoolean(lv.str().s == rv.str().s)
		case ttNotEqual:
			return ValueFromBoolean(lv.str().s != rv.str().s)
		default:
			panic(NewSyntaxError("not supported operator on two strings"))
		}
	}

	if op == ttLogicalAnd {
		return ValueFromBoolean(
			b.left.Evaluate(ctx).Truth(ctx) &&
				b.right.Evaluate(ctx).Truth(ctx),
		)
	} else if op == ttLogicalOr {
		lv = b.left.Evaluate(ctx)
		if lv.Truth(ctx) {
			return lv
		}
		return b.right.Evaluate(ctx)
	}

	if lt == vtBuiltin && rt == vtBuiltin {
		p1 := reflect.ValueOf(lv.builtin().fn).Pointer()
		p2 := reflect.ValueOf(rv.builtin().fn).Pointer()
		switch op {
		case ttEqual:
			return ValueFromBoolean(p1 == p2)
		case ttNotEqual:
			return ValueFromBoolean(p1 != p2)
		default:
			panic(NewSyntaxError("not supported operator on two builtins"))
		}
	}

	panic(NewSyntaxError("unknown binary operator and operands"))
}

// TernaryExpression is the `cond ? left : right` expression.
type TernaryExpression struct {
	cond  Expression
	left  Expression
	right Expression
}

// NewTernaryExpression news a ternary expression.
func NewTernaryExpression(cond, left, right Expression) *TernaryExpression {
	return &TernaryExpression{
		cond:  cond,
		left:  left,
		right: right,
	}
}

// Evaluate implements Expression.
func (t *TernaryExpression) Evaluate(ctx *Context) Value {
	if t.cond.Evaluate(ctx).Truth(ctx) {
		return t.left.Evaluate(ctx)
	}
	return t.right.Evaluate(ctx)
}

// NewExpression is the `new Type()` expression.
type NewExpression struct {
	Type string
	Args *Arguments
}

// Evaluate implements Expression.
func (e *NewExpression) Evaluate(ctx *Context) Value {
	ctorValue := ctx.MustFind(e.Type, true)
	if !ctorValue.isConstructor() {
		panic(NewSyntaxError("%s is not a constructor", e.Type))
	}
	args := e.Args.EvaluateAll(ctx)
	obj := ctorValue.constructable().Ctor(args.values...)
	return ValueFromObject(obj)
}

// AssignmentExpression is an assignment expression.
// Notice: In tao, assignment is not actually an expression.
type AssignmentExpression struct {
	left  Expression
	right Expression
}

// NewAssignmentExpression news an assignment expression.
func NewAssignmentExpression(left Expression, right Expression) *AssignmentExpression {
	return &AssignmentExpression{
		left:  left,
		right: right,
	}
}

// Evaluate implements Expression.
func (a *AssignmentExpression) Evaluate(ctx *Context) Value {
	assigner, ok := a.left.(Assigner)
	if !ok {
		val := a.left.Evaluate(ctx)
		panic(NewNotAssignableError(val))
	}
	value := a.right.Evaluate(ctx)
	assigner.Assign(ctx, value)
	return value
}

// Parameters is a collection of function parameters.
type Parameters struct {
	names []string
}

// NewParameters news
func NewParameters(names ...string) *Parameters {
	p := &Parameters{
		names: names,
	}
	return p
}

// Len returns the count of parameters.
func (p *Parameters) Len() int {
	return len(p.names)
}

// PutParam adds a parameter.
func (p *Parameters) PutParam(name string) {
	p.names = append(p.names, name)
}

// BindArguments assigns actual arguments.
// un-aligned parameters and arguments are set to nil.
func (p *Parameters) BindArguments(ctx *Context, args ...Value) {
	for index, name := range p.names {
		var arg Value
		if index < len(args) {
			arg = args[index]
		}
		ctx.AddSymbol(name, arg)
	}
}

// EvaluatedFunctionExpression is the result of a FunctionExpression.
// The result is the closure and the expr itself.
//  Evaluate(FunctionExpression) -> EvaluatedFunctionExpression
//  Execute(EvaluatedFunctionExpression) -> Execute(FunctionExpression, this)
type EvaluatedFunctionExpression struct {
	this *Context // this is the scope where the function expression is defined
	fn   *FunctionExpression
}

// Execute evaluates the function expression within closure.
// It implements Callable.
func (e *EvaluatedFunctionExpression) Execute(ctx *Context, args *Values) Value {
	ctx.SetParent(e.this) // this is how closure works
	return e.fn.Execute(ctx, args)
}

// FunctionExpression is
type FunctionExpression struct {
	name   string
	params *Parameters
	body   *BlockStatement
}

// Evaluate is
func (f *FunctionExpression) Evaluate(ctx *Context) Value {
	value := ValueFromFunction(f, ctx)
	// a lambda function or an anonymous function doesn't have a name.
	if f.name != "" {
		ctx.AddSymbol(f.name, value)
	}
	return value
}

// Execute executes function statements.
// It implements Callable.
func (f *FunctionExpression) Execute(ctx *Context, args *Values) Value {
	f.params.BindArguments(ctx, args.values...)
	f.body.Execute(ctx)
	if ctx.hasret {
		return ctx.retval
	}
	return ValueFromNil()
}

// Arguments is the collection of arguments for function call.
type Arguments struct {
	exprs []Expression
}

// Len returns the length of arguments.
func (a *Arguments) Len() int {
	return len(a.exprs)
}

// PutArgument adds an argument.
func (a *Arguments) PutArgument(expr Expression) {
	a.exprs = append(a.exprs, expr)
}

// EvaluateAll evaluates all values of arguments.
func (a *Arguments) EvaluateAll(ctx *Context) Values {
	args := Values{}
	for _, expr := range a.exprs {
		args.values = append(args.values, expr.Evaluate(ctx))
	}
	return args
}

// IndexExpression is
// obj.key    -> key: identifier whose name is "key"
// obj[key]   -> key: expression that returns string
// formally: obj should be `indexable', which supports
// syntaxes like: "str".len(), or: 123.str()
type IndexExpression struct {
	indexable Expression
	key       Expression
}

// Evaluate implements Expression.
func (i *IndexExpression) Evaluate(ctx *Context) Value {
	key := i.key.Evaluate(ctx)
	indexable := i.indexable.Evaluate(ctx)

	// both obj.key, obj[0] are correct.
	// so, we need to query both interfaces.
	obj, _ := indexable.value.(IObject)
	arr, _ := indexable.value.(IArray)

	// get property
	if key.Type == vtString && obj != nil {
		return obj.GetProp(key.str().s)
	}

	// get element
	if key.Type == vtNumber && arr != nil {
		return arr.GetElem(key.number())
	}

	if obj == nil && arr == nil {
		panic(NewNotIndexableError(indexable))
	}

	if obj != nil && key.Type != vtString {
		panic(NewKeyTypeError(key))
	}
	if arr != nil && key.Type != vtNumber {
		panic(NewKeyTypeError(key))
	}

	panic("won't go here")
}

// Assign implements Assigner.
func (i *IndexExpression) Assign(ctx *Context, val Value) {
	value := i.indexable.Evaluate(ctx)
	obj, ok1 := value.value.(IObject)
	arr, ok2 := value.value.(IArray)
	if !ok1 && !ok2 {
		panic(NewNotAssignableError(value))
	}
	key := i.key.Evaluate(ctx)
	if key.isString() && obj != nil {
		obj.SetProp(key.str().s, val)
		return
	}
	if key.isNumber() && arr != nil {
		arr.SetElem(key.number(), val)
		return
	}
	panic(NewKeyTypeError(key))
}

// CallExpression wraps a call.
type CallExpression struct {
	Callable Expression
	Args     Arguments
}

// NewCallExpression news a call expression.
func NewCallExpression(callable Expression, args ...Expression) *CallExpression {
	c := &CallExpression{}
	c.Callable = callable
	c.Args.exprs = args
	return c
}

// CallFunc calls user function.
func CallFunc(ctx *Context, callable Expression, args ...Expression) Value {
	c := NewCallExpression(callable, args...)
	return c.Evaluate(ctx)
}

// Evaluate implements Expression.
// It calls the callable.
func (f *CallExpression) Evaluate(ctx *Context) Value {
	callable := f.Callable.Evaluate(ctx)

	if callable.Type == vtVariable {
		callable = callable.Evaluate(ctx)
	}

	if callable.isCallable() {
		newCtx := NewContext("", nil)
		args := f.Args.EvaluateAll(ctx)
		return callable.callable().Execute(newCtx, &args)
	}

	panic(NewNotCallableError(callable))
}

// ObjectExpression is the object literal expression.
type ObjectExpression struct {
	props map[string]Expression
}

// NewObjectExpression news an object literal expression.
func NewObjectExpression() *ObjectExpression {
	return &ObjectExpression{
		props: make(map[string]Expression),
	}
}

// Evaluate implements Expression.
func (o *ObjectExpression) Evaluate(ctx *Context) Value {
	obj := NewObject()
	for k, v := range o.props {
		obj.props[k] = v.Evaluate(ctx)
	}
	return ValueFromObject(obj)
}

// ArrayExpression is the array literal expression.
type ArrayExpression struct {
	elements []Expression
}

// NewArrayExpression news an array literal expression.
func NewArrayExpression() *ArrayExpression {
	return &ArrayExpression{}
}

// Evaluate implements Expression.
func (a *ArrayExpression) Evaluate(ctx *Context) Value {
	arr := NewArray()
	for _, element := range a.elements {
		arr.PushElem(element.Evaluate(ctx))
	}
	return ValueFromObject(arr)
}
