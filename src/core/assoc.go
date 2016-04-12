package core

// type A int
// type B string

// type Object struct {
// 	a A
// 	b B
// }

// type ASSOC []Object

// func aDomain(assoc ASSOC) []A {
// 	var a = []A{}
// 	for _,obj := range assoc {
// 		a = append(a, obj.a)
// 	}
// 	return a
// }
// // obj := ASSOC{{1,"datta"},{2,"mahesh"}}
// // fmt.Println(obj)
// // fmt.Println(aDomain(obj))

// func aRange(assoc ASSOC) []B {
// 	var b = []B{}
// 	for _,obj := range assoc {
// 		b = append(b, obj.b)
// 	}
// 	return b
// }

// func aLookup(assoc ASSOC, sear Object) B {
// 	for _,obj := range assoc {
// 		if obj.a == sear.a  {
// 			return obj.b
// 		}
// 	}
// 	return "" //Default Value: null string
// }
// //ab := Object{2,"mahesh"
// //fmt.Println(aLookup(obj,ab))

// func aEmpty(assoc ASSOC) bool{
// 	if len(assoc) == 0 {
// 		return false;
// 	}
// 	return true;
// }

// func elem(name B, assoc ASSOC) A {
// 	for _,obj := range assoc {
// 		if obj.b == name  {
// 			return obj.a
// 		}
// 	}
// 	return -1 //Default Value: null string
// }