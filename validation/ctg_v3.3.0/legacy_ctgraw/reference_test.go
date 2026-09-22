package ctgraw
import("testing";"os")
func TestReference(t *testing.T){
 base:=os.Getenv("CTG_REFERENCE_BOOK");out:=os.Getenv("CTG_REFERENCE_OUT")
 if base==""||out==""{t.Skip("set CTG_REFERENCE_BOOK and CTG_REFERENCE_OUT")}
 res,e:=BuildNeutral(base,base+".ctg",base+".cto",base+".ctb",out,100);if e!=nil{t.Fatal(e)}
 t.Logf("%+v",res)
}
