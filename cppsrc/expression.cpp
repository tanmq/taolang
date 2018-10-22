#include "expression.h"
#include "statement.h"
#include "value.h"
#include "tokenizer.h"
#include "error.h"
#include "object.h"

namespace taolang {

// done
Value* UnaryExpression::Evaluate(Context* ctx) {
    auto value = _expr->Evaluate(ctx);
    switch(_op) {
    case ttLogicalNot:
        return Value::fromBoolean(!value->truth(ctx));
    case ttAddition:
        if(value->type != ValueType::Number) {
            throw TypeError("+value is invalid");
        }
        return Value::fromNumber(+value->number());
    case ttSubtraction:
        if(value->type != ValueType::Number) {
            throw TypeError("-value is invalid");
        }
        return Value::fromNumber(+value->number());
    case ttBitXor:
        if(value->type != ValueType::Number) {
            throw TypeError("^value is invalid");
        }
        return Value::fromNumber(~value->number());
    default:
        break;
    }
    throw SyntaxError("unknown unary operator: %s", Token(_op).string().c_str());
}

Value* IncrementExpression::Evaluate(Context* ctx) {
    auto oldval = _expr->Evaluate(ctx);
    if(oldval->isNumber()) {
        auto newnum = int64_t(0);
        switch(_op) {
        case ttIncrement:
            newnum = oldval->number() + 1;
            break;
        case ttDecrement:
            newnum = oldval->number() - 1;
            break;
        default:
            throw Error();
        }
        auto newval = Value::fromNumber(newnum);
        _expr->Assign(ctx, newval);
        return _prefix ? newval : oldval;
    }
    throw NotAssignableError(
        "`%s' is not assignable",
        oldval->ToString().c_str()
    );
}

Value* BinaryExpression::Evaluate(Context* ctx) {
    Value *lv, *rv;
    if(_op != ttLogicalAnd && _op != ttLogicalOr){
        lv = _left->Evaluate(ctx);
        rv = _right->Evaluate(ctx);
    }

    auto lt=lv->type, rt = rv->type;

    if(lt == ValueType::Nil && rt == ValueType::Nil) {
        switch(_op) {
        case ttEqual:
            return Value::fromBoolean(true);
        case ttNotEqual:
            return Value::fromBoolean(false);
        default:
            break;
        }
    }

    if(lt == ValueType::Boolean && rt == ValueType::Boolean) {
        switch(_op) {
        case ttEqual:
            return Value::fromBoolean(lv->boolean() == rv->boolean());
        case ttNotEqual:
            return Value::fromBoolean(lv->boolean() != rv->boolean());
        default:
            break;
        }
    }

    if(lt == ValueType::Number && rt == ValueType::Number){
        switch(_op){
        case ttAddition:
            return Value::fromNumber(lv->number() + rv->number());
        case ttSubtraction:
            return Value::fromNumber(lv->number() - rv->number());
        case ttMultiply:
            return Value::fromNumber(lv->number() * rv->number());
        case ttDivision:
            if(rv->number() == 0) {
                throw TypeError("divide by zero");
            }
            return Value::fromNumber(lv->number() / rv->number());
        case ttGreaterThan:
            return Value::fromBoolean(lv->number() > rv->number());
        case ttGreaterThanOrEqual:
            return Value::fromBoolean(lv->number() >= rv->number());
        case ttLessThan:
            return Value::fromBoolean(lv->number() < rv->number());
        case ttLessThanOrEqual:
            return Value::fromBoolean(lv->number() <= rv->number());
        case ttEqual:
            return Value::fromBoolean(lv->number() == rv->number());
        case ttNotEqual:
            return Value::fromBoolean(lv->number() != rv->number());
        case ttPercent:
            return Value::fromNumber(lv->number() % rv->number());
        case ttStarStar:
            // TODO precision lost
            // val := math.Pow(float64(lv->number()), float64(rv->number()))
            //return Value::fromNumber(int(val))
        case ttLeftShift:
            return Value::fromNumber(lv->number() << uint(rv->number()));
        case ttRightShift:
            return Value::fromNumber(lv->number() >> uint(rv->number()));
        case ttBitAnd:
            return Value::fromNumber(lv->number() & rv->number());
        case ttBitOr:
            return Value::fromNumber(lv->number() | rv->number());
        case ttBitXor:
            return Value::fromNumber(lv->number() ^ rv->number());
        case ttBitAndNot:
            return Value::fromNumber(lv->number() &~ rv->number());
        default:
            break;
        }
    }

    if(lt == ValueType::String && rt == ValueType::String){
        switch(_op) {
        case ttAddition:
            return Value::fromString(lv->str + rv->str);
        case ttEqual:
            return Value::fromBoolean(lv->str == rv->str);
        case ttNotEqual:
            return Value::fromBoolean(lv->str != rv->str);
        default:
            throw SyntaxError("not supported operator on two strings");
        }
    }

    if(_op == ttLogicalAnd) {
        return Value::fromBoolean(
            _left->Evaluate(ctx)->truth(ctx) &&
                _right->Evaluate(ctx)->truth(ctx)
        );
    } else if(_op == ttLogicalOr) {
        lv = _left->Evaluate(ctx);
        if(lv->truth(ctx)) {
            return lv;
        }
        return _right->Evaluate(ctx);
    }

    if(lt == ValueType::Builtin && rt == ValueType::Builtin) {
        auto p1 = lv->builtin()->_func;
        auto p2 = rv->builtin()->_func;
        switch(_op) {
        case ttEqual:
            return Value::fromBoolean(p1 == p2);
        case ttNotEqual:
            return Value::fromBoolean(p1 != p2);
        default:
            throw SyntaxError("not supported operator on two builtins");
        }
    }

    throw SyntaxError("unknown binary operator and operands");
}

Value* TernaryExpression::Evaluate(Context* ctx) {
    return cond->Evaluate(ctx)->truth(ctx)
        ? left->Evaluate(ctx)
        : right->Evaluate(ctx)
        ;
}

Value* NewExpression::Evaluate(Context* ctx) {
    throw Error("new()");
}

Value* AssignmentExpression::Evaluate(Context* ctx) {
    auto val = _expr->Evaluate(ctx);
    _left->Assign(ctx, val);
    return val;
}

Value* EvaluatedFunctionExpression::Execute(Context* ctx, Values* args) {
    ctx->SetParent(_closure);
    return _func->Execute(ctx, args);
}

Value* FunctionExpression::Evaluate(Context* ctx) {
    auto val = Value::fromFunction(this, ctx);
    if(!_name.empty()) {
        ctx->AddSymbol(_name, val);
    }
    return val;
}

Value* FunctionExpression::Execute(Context* ctx, Values* args) {
    _params.BindArguments(ctx, args);
    _body->Execute(ctx);
    if(ctx->_hasRet) {
        return ctx->_retVal;
    }
    return Value::fromNil();
}

Value* ObjectExpression::Evaluate(Context* ctx) {
    auto obj = Object::New();
    for(auto& prop : _props) {
        obj->SetKey(prop.first, prop.second->Evaluate(ctx));
    }
    return Value::fromObject(obj);
}

Value* ArrayExpression::Evaluate(Context* ctx) {
    return Value::fromNil();
}

Value* CallExpression::Evaluate(Context* ctx) {
    auto callable = _callable->Evaluate(ctx);
    if(callable->type == ValueType::Variable) {
        callable = callable->Evaluate(ctx);
    }
    if(!callable->isCallable()) {
        throw NotCallableError(
            "`%s' is not callable",
            callable->ToString().c_str()
        );
    }
    auto newCtx = new Context(nullptr);
    auto args = _args.EvaluateAll(ctx);
    return callable->callable()->Execute(newCtx, args);
}

Value* CallFunc(Context* ctx, IExpression* callable, Arguments* args) {
    auto ce = new CallExpression();
    ce->_callable = callable;
    if(args != nullptr) {
        for(int i = 0; i < args->Size(); i++) {
            ce->_args.Put(args->Get(i));
        }
    }
    return ce->Evaluate(ctx);
}

Value* IndexExpression::Evaluate(Context* ctx) {
    auto indexable = _indexable->Evaluate(ctx);
    auto key = _key->Evaluate(ctx);

    IObject* obj = nullptr;
    IArray* arr = nullptr;

    if(indexable->isObject()) {
        obj = indexable->object();
    }
    if(indexable->isArray()) {
        arr = indexable->array();
    }

    if(key->isString() && obj != nullptr) {
        return obj->GetKey(key->string());
    }
    if(key->isNumber() && arr != nullptr) {
        return arr->GetElem(key->number());
    }

    throw TypeError(
        "cannot use `%s' (type: %s) as key",
        key->ToString().c_str(),
        key->TypeName().c_str()
    );
}

void IndexExpression::Assign(Context* ctx, Value* value) {
    auto indexable = _indexable->Evaluate(ctx);
    auto key = _key->Evaluate(ctx);

    IObject* obj = nullptr;
    IArray* arr = nullptr;

    if(indexable->isObject()) {
        obj = indexable->object();
    }
    if(indexable->isArray()) {
        arr = indexable->array();
    }

    if(!obj && !arr) {
        throw NotAssignableError(
            "`%s' is not assignable",
            indexable->ToString().c_str()
        );
    }

    if(key->isString() && obj != nullptr) {
        return obj->SetKey(key->string(), value);
    }
    if(key->isNumber() && arr != nullptr) {
        return arr->SetElem(key->number(), value);
    }

    throw TypeError(
        "cannot use `%s' (type: %s) as key",
        key->ToString().c_str(),
        key->TypeName().c_str()
    );
}

}
