; *************** Pushglobal {{.Name}} ***************
; allocate global node on the heap
%pglobal{{.Ninstr}} = call i64*(i64, void()*)* @hAllocGlobal(i64 {{.Arity}}, void()* @_{{.Name}})

; store function cell address on the stack
call void(i64*)* @push(i64* %pglobal{{.Ninstr}})
call i32 @puts(i8* getelementptr inbounds ([12 x i8]* @pushglobalmsg, i32 0, i32 0))

