package x

//go:generate craft macro xx ww oo ll x2=xx

// TODO: allow macro declaration on a var definition that it's type is unexported
// TODO: check if the type is exported then do not allow a macro declaration on a var definition
// TODO: throw an error if the type is unexported and there is macro declaration on it
type name string

// #macro.macro
//
/* #macro.second_macro */
/* #macro.third_macro_line0
   #macro.third_macro_line1
*/
// #macro.fourth(`hello`)
/*
   #macro.fifth_macro_line0
   #macro.fifth_macro_line1
*/
// #macro.sixth
/*!
  #macro.seventh_macro_line0
  #macro.seventh_macro_line1*/
// #macro.eighth
/* #macro.ninth_macro_line0
   #macro.ninth_macro_line1*/
// #x2.yy
/*#ww.zz
  #     oo.pp
      #ll.ll*/
// #macro.tenth
const Name__ name = ""

// #macro.Macro(`hey`)
/*#x2.x2*/
// #ll.LL(`lol`)
type X int
