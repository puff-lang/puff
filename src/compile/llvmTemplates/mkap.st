; *************** Mkap ***************
%fun{{.Ninstr}} = call i64*()* @pop()
%arg{{.Ninstr}} = call i64*()* @pop()

; create application node on the heap
%ap{{.Ninstr}} = call i64*(i64*, i64*)* @hAllocAp(i64* %fun{{.Ninstr}}, i64* %arg{{.Ninstr}})

; push the newly allocated appliation node address onto the stack
call void(i64*)* @push(i64* %ap{{.Ninstr}})
call i32 @puts(i8* getelementptr inbounds ([12 x i8]* @mkapmsg, i32 0, i32 0))

