; *************** CasejumpConstr ***************
%ptop.{{.Ninstr}} = call i64**()* @getTopPtr()
%top.{{.Ninstr}} = load i64** %ptop.{{.Ninstr}}
%ptag.{{.Ninstr}} = call i64*(i64*)* @getConstrTagPtr(i64* %top.{{.Ninstr}})
%tag.{{.Ninstr}} = load i64* %ptag.{{.Ninstr}}

switch i64 %tag.{{.Ninstr}}, label %DEFAULT.{{.Ninstr}} [ {{.Branches}} ]

{{.Alts}}

DEFAULT.{{.Ninstr}}:
    br label %END_CASE.{{.Ninstr}}

END_CASE.{{.Ninstr}}:

