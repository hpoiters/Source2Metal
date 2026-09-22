import chess,chess.pgn,sys,json,time,struct,hashlib
from pathlib import Path

def inspect(path):
 start=time.perf_counter();edges=set();positions=set();records=plies=0;first=set()
 with open(path,encoding='utf-8') as f:
  while (g:=chess.pgn.read_game(f)) is not None:
   if g.errors:raise ValueError((path,records,g.errors))
   b=g.board()
   for m in g.mainline_moves():
    if m not in b.legal_moves:raise ValueError((path,records,'illegal',m))
    key=b._transposition_key();positions.add(key);edges.add((key,m.uci()))
    if b.fullmove_number==1 and b.turn==chess.WHITE:first.add(m.uci())
    b.push(m);plies+=1
   positions.add(b._transposition_key());records+=1
 return dict(records=records,plies=plies,positions=len(positions),edges=len(edges),first_moves=sorted(first),seconds=time.perf_counter()-start),edges
if __name__=='__main__':
 a,ae=inspect(sys.argv[1]);b,be=inspect(sys.argv[2]);out={'old':a,'new':b,'new_only_edges':len(be-ae),'old_only_edges':len(ae-be),'shared_edges':len(ae&be)}
 Path(sys.argv[3]).write_text(json.dumps(out,indent=2));print(json.dumps(out),flush=True)
