package main

type Statement interface {
	Execute(ctx *Context)
}

type VariableDefinitionStatement struct {
	Name string
	Expr Expression
}

func (v *VariableDefinitionStatement) Execute(ctx *Context) {
	ctx.AddValue(v.Name, v.Expr.Evaluate(ctx))
}

type VariableAssignmentStatement struct {
	Name string
	Expr Expression
}

func (v *VariableAssignmentStatement) Execute(ctx *Context) {
	// TODO
}

type FunctionDefinitionStatement struct {
	name string
	expr *FunctionDefinitionExpression
}

func (f *FunctionDefinitionStatement) Execute(ctx *Context) {
	ctx.AddValue(f.name, ValueFromFunction(f.name, f.expr))
}

type ReturnStatement struct {
	expr  Expression
	value *Value
}

func (r *ReturnStatement) Execute(ctx *Context) {
	r.value = r.expr.Evaluate(ctx)
}

type ExpressionStatement struct {
	expr Expression
}

func (r *ExpressionStatement) Execute(ctx *Context) {
	value := r.expr.Evaluate(ctx)
	_ = value // drop expr value
}
