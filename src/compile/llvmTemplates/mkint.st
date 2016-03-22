; *************** MkInt ***************
%n.{{.Ninstr}} = call i64()* @popV()

; alloc num node on the heap
%num.{{.Ninstr}} = call i64*(i64)* @hAllocNum(i64 %n.{{.Ninstr}})

; push address onto the stack
call void(i64*)* @push(i64* %num.{{.Ninstr}})

