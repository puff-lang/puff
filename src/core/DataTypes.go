package core

import (
	// "strconv"
	// "fmt"
)

//-----------------------------Abstract Data Types----------------------------------
// type Tag int
// type  Arity int
type NameConstrMapping struct {
	Name string
	Const Constructor
}

var trueTag int = int(1)
var falseTag int = int(0)
var consTag int = int(3)
var nilTag int = int(2)
var initialTag int = int(4)
var undefinedTag int = int(-1)

//-----------------------------Primitive ADT's------------------------------------------
type Constructor struct {
	Name string
	Tag int
	Arity int
}

type DataType struct {
	Name string
	Consts Constructors
}

type Constructors []Constructor

type DataTypes []DataType

var primitiveADTs = DataTypes{
	DataType{"Bool", Constructors{Constructor{"True", trueTag, 0}, Constructor{"False", falseTag, 0}}},
	DataType{"List", Constructors{Constructor{"Nil", nilTag, 0}, Constructor{"Cons", consTag, 2}}},
	DataType{"Tuple0", Constructors{Constructor{"Tuple0", undefinedTag, 0}}},
}





