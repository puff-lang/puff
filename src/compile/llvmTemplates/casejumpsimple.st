; *************** CasejumpSimple ***************
%ptop.{{.Ninstr}} = call i64**()* @getTopPtr()
%top.{{.Ninstr}} = load i64** %ptop.{{.Ninstr}}
%pnum.{{.Ninstr}} = call i64*(i64*)* @getNumPtr(i64* %top.{{.Ninstr}})
%num.{{.Ninstr}} = load i64* %pnum.{{.Ninstr}}

switch i64 %num.{{.Ninstr}}, label %ALT-1.{{.Ninstr}} [ {{.Branches}} ]

{{.Alts}}

DEFAULT.{{.Ninstr}}:
    br label %END_CASE.{{.Ninstr}}

END_CASE.{{.Ninstr}}:

