Currently generic types are not supported!

In order to support generice types, Craft must include source code information beside reflection data:

```golang
func Macro(input string, data TypeData) (string, error)
```

`TypeData`	type contain both source code and reflection data:

```golang
type TypeData struct {
    TypeOf reflect.TypeOf
    Source *ast.TypeSpec
}
```

Macro owners should check whether a type is generic in order to get more insight from the types.

---

For Craft in order to inintialize a generic type in the generated program, we should think more about it:

If there was not a solution then the user itself should do this for us:

```golang
type GenericeType[T constraint.SomeExternalConstraint] struct {}

// #macro.Macro
var Variable GenericeType[SomeTypeThatSatisfiesExternalConstraint]
```

---

Should all macro declaration have to be on a var definition? it makes it general:

```golang
type Foo string

// #macro.Bar
/*#macro.Baz*/
var Foo__ Foo
```

No! bc we need source code *ast.TypeSpec data