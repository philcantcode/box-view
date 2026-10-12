import {test, expect} from '@playwright/test';

test('nested model, collapse, evidence, query, history and API editing', async ({page}) => {
 const errors = [];page.on('pageerror', error => errors.push(error.message));
 await page.goto('/');
 await expect(page.locator('#status')).toContainText('Revision 1');
 await expect(page.getByRole('button', {name:'Message database · datastore', exact:true})).toBeVisible();
 const storage = page.getByRole('region', {name:'Storage', exact:true});
 // Native section landmarks have a region role when named.
 await storage.getByRole('button', {name:'Collapse Storage', exact:true}).click();
 await expect(page.getByRole('button', {name:'Message database · datastore', exact:true})).toHaveCount(0);
 await page.getByRole('button', {name:'Sync → writes · 1 assertion → Storage', exact:true}).click();
 await expect(page.locator('#inspector')).toContainText('sync-writes');
 await expect(page.locator('#inspector')).toContainText('events:84-91');
 await page.locator('#lens').selectOption('writes');
 await expect(page.locator('.edges button')).toHaveCount(1);
 await page.getByText('Query the model', {exact:true}).click();
 await page.locator('#kind').selectOption('boundary');
 await page.getByRole('button', {name:'Query',exact:true}).click();
 await expect(page.locator('#inspector')).toContainText('knowledgeCompleteness');
 await page.getByText('Apply an atomic patch', {exact:true}).click();
 await page.getByRole('button', {name:'Apply patch', exact:true}).click();
 await expect(page.locator('#status')).toContainText('Revision 2');
 await expect(page.getByRole('button', {name:'Synchronizer · component',exact:true})).toBeVisible();
 await page.getByRole('button', {name:'Apply patch', exact:true}).click();
 await expect(page.locator('#status')).toContainText('revision conflict');
 await page.locator('#revision').fill('1');await page.locator('#refresh').click();
 await expect(page.getByRole('button', {name:'Sync · component',exact:true})).toBeVisible();
 expect(errors).toEqual([]);
});

test('colours, descriptions and spans respect responsive row configuration', async ({page}) => {
 await page.setViewportSize({width:1500,height:1000});
 const snapshot=await (await page.request.get('/api/snapshot')).json();
 await page.request.post('/api/apply',{data:{expectedRevision:snapshot.revision,actor:'test',reason:'Box description',operations:[{action:'update',entity:{...snapshot.entities.storage,description:'Storage description <script>literal text</script>'}}]}});
 await page.goto('/');await expect(page.locator('#status')).toContainText('Revision');
 await page.getByText('Box appearance',{exact:true}).click();
 await page.locator('#appearance').fill(JSON.stringify({styles:{storage:{colour:'#4f83cc',span:2},sync:{colour:'#b78aff',span:1}}}));
 await page.getByRole('button',{name:'Update appearance',exact:true}).click();
 const storage=page.locator('[data-entity-id="storage"]'),sync=page.locator('[data-entity-id="sync"]');
 await expect(storage.locator('.box-description').first()).toHaveText('Storage description <script>literal text</script>');
 await expect(storage).toHaveCSS('border-left-color','rgb(79, 131, 204)');
 await expect.poll(async()=>{const a=await storage.boundingBox(),b=await sync.boundingBox();return a&&b ? a.width/b.width : 0;}).toBeGreaterThan(1.8);
 await page.locator('#max-per-row').fill('1');await page.locator('#refresh').click();
 await expect.poll(async()=>{const a=await storage.boundingBox(),b=await sync.boundingBox();return a&&b ? Math.abs(a.width-b.width) : Infinity;}).toBeLessThan(1);
 await page.setViewportSize({width:375,height:900});
 await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
});

test('relationship dots reveal arrows on hover/focus and pin exact evidence',async({page})=>{
 await page.goto('/');await expect(page.locator('.relationship-dot')).toHaveCount(4);
 await expect(page.locator('[data-entity-id="sync"]>.box-connections>.relationship-dot')).toHaveCount(2);
 await expect(page.locator('[data-entity-id="db"]>.box-connections>.relationship-dot')).toHaveCount(1);
 for(const id of ['sync','db','storage-api']){
   const box=page.locator(`[data-entity-id="${id}"]`),dot=box.locator(':scope>.box-connections>.relationship-dot').first();
   const r=await box.boundingBox(),d=await dot.boundingBox();expect(d.x-r.x).toBeLessThan(25);expect(r.y+r.height-d.y-d.height).toBeLessThan(25);
 }
 await page.mouse.move(0,0);await page.locator('h1').click();
 await expect(page.locator('.edge-arrow.active')).toHaveCount(0);
 await page.locator('[data-entity-id="sync"]>.box-head .box-title').hover();
 await expect(page.locator('.edge-arrow.active')).toHaveCount(2);
 await expect(page.locator('.edge-arrow.active').first()).toHaveCSS('opacity','1');
 await expect(page.locator('[data-entity-id="db"]')).toHaveClass(/connected/);
 await page.mouse.move(0,0);await expect(page.locator('.edge-arrow.active')).toHaveCount(0);
 const writes=page.getByRole('button',{name:/Reveal relationship: .*writes/}).first();
 const writeMarkers=page.getByRole('button',{name:/Reveal relationship: .*writes/});
 expect(await writeMarkers.locator('.dot-core').evaluateAll(nodes=>nodes.map(n=>getComputedStyle(n).backgroundColor))).toEqual(['rgb(243, 181, 98)','rgb(243, 181, 98)']);
 await writes.focus();await expect(page.locator('.edge-arrow.active')).toHaveCount(1);
 await writes.press('Enter');await expect(writes).toHaveAttribute('aria-pressed','true');
 await expect(writeMarkers.nth(1)).toHaveAttribute('aria-pressed','true');
 await expect(page.locator('#inspector')).toContainText('sync-writes');
 await page.locator('#refresh').focus();await expect(page.locator('.edge-arrow.active')).toHaveCount(1);
 await writes.focus();await writes.press('Escape');await page.locator('#refresh').focus();
 await expect(page.locator('.edge-arrow.active')).toHaveCount(0);
 const before=await writes.boundingBox();await page.setViewportSize({width:400,height:1000});
 await expect.poll(async()=>{const r=await writes.boundingBox();return r&&Math.abs(r.x-before.x)>5;}).toBe(true);
 await page.getByRole('region',{name:'Storage',exact:true}).getByRole('button',{name:'Collapse Storage',exact:true}).click();
 await expect(page.getByRole('button',{name:/Reveal relationship: .*writes.*Storage$/})).toHaveCount(2);
});
