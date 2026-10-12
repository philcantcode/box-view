import test from 'node:test';
import assert from 'node:assert/strict';
globalThis.HTMLElement = class {};
globalThis.customElements = {define() {}};
const {forest, BoxClient, connection} = await import('./box-view.js');
test('directional paths, parallel midpoint spacing and self loops',()=>{
 const a={x:0,y:0,width:100,height:40},b={x:200,y:0,width:100,height:40};
 const route=connection(a,b);
 assert.deepEqual(route.start,{x:100,y:20});assert.deepEqual(route.end,{x:200,y:20});assert.deepEqual(route.dot,{x:150,y:20});
 assert.equal(connection(a,b,.5).dot.y-connection(a,b,-.5).dot.y,24);
 const loop=connection(a,a,0,true);assert.notDeepEqual(loop.start,loop.end);assert.ok(!loop.path.includes('NaN'));
});
test('nest by canonical identity, independent of labels and input order', () => {
 const roots = forest([{entity:{id:'child',label:'same'},parent:'parent'},{entity:{id:'parent',label:'same'}}]);
 assert.equal(roots.length,1);assert.equal(roots[0].children[0].entity.id,'child');
});
test('client uses revision pinned explanation and surfaces conflicts', async () => {
 const original = globalThis.fetch;
 try {
  globalThis.fetch = async (url, options) => {assert.equal(url,'http://localhost:8080/api/explain?revision=12');assert.equal(options.headers.Authorization,'Bearer token');assert.deepEqual(JSON.parse(options.body),{ids:['a']});return {ok:true,text:async()=>'{"revision":12}'};};
  assert.equal((await new BoxClient('http://localhost:8080/','token').explain(['a'],12)).revision,12);
  globalThis.fetch = async () => ({ok:false,status:409,text:async()=>'{"error":"revision conflict"}'});
  await assert.rejects(new BoxClient().apply({}),/revision conflict/);
  globalThis.fetch = async () => ({ok:false,status:401,text:async()=>'unauthorized'});
  await assert.rejects(new BoxClient().snapshot(),/HTTP 401: unauthorized/);
 } finally {globalThis.fetch = original;}
});
