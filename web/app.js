import {BoxClient} from './box-view.js';
const $ = id => document.getElementById(id);
const viewer = document.querySelector('box-view');
const collapsed = new Set(); let current; let generation = 0;
const client = () => new BoxClient('', $('token').value);
const revision = () => $('revision').value === '' ? undefined : Number($('revision').value);
const filter = () => ({hierarchy: 'architecture', scope: {target: $('target').value}, predicates: $('lens').value ? [$('lens').value] : [], dispositions: ['accepted'], applicabilities: ['applicable']});
const inspect = value => { $('inspector').textContent = JSON.stringify(value, null, 2); };
async function guard(action) {try {await action();} catch (error) {$('status').textContent = error.message;}}
async function refresh() {
  const request = ++generation;
  const appearance = JSON.parse($('appearance').value);
  const layout = {...appearance.layout};
  if ($('max-per-row').value) layout.maxBoxesPerRow = Number($('max-per-row').value);
  const result = await client().view({filter: filter(), collapsed: [...collapsed], styles: appearance.styles, layout}, revision());
  if (request !== generation) return;
  current = result;
  viewer.projection = current;
  $('status').textContent = `Revision ${current.revision} · ${current.boxes.length} visible components · ${current.edges.length} relationship groups`;
  if (!$('patch').value) $('patch').value = JSON.stringify({expectedRevision: current.revision, actor: 'human', reason: 'Rename Sync', operations: [{action: 'update', entity: {id: 'sync', kind: 'component', label: 'Synchronizer'}}]}, null, 2);
}
viewer.addEventListener('toggle-box', event => guard(async () => {if (collapsed.has(event.detail)) collapsed.delete(event.detail);else collapsed.add(event.detail);await refresh();}));
viewer.addEventListener('select-edge', event => guard(async () => inspect(await client().explain(event.detail.members, current.revision))));
viewer.addEventListener('select-entity', event => guard(async () => {const s = await client().snapshot(current.revision);inspect({entity: s.entities[event.detail], boundary: await client().query({kind: 'boundary', entityId: event.detail, filter: filter()}, current.revision)});}));
$('refresh').onclick = () => guard(refresh);
for (const id of ['lens', 'target', 'revision', 'max-per-row']) $(id).addEventListener('change', () => guard(refresh));
$('appearance-apply').onclick = () => guard(refresh);
$('apply').onclick = () => guard(async () => {const commit = await client().apply(JSON.parse($('patch').value));inspect(commit);$('revision').value = '';await refresh();});
$('diff').onclick = () => guard(async () => {inspect(await client().request(`diff?from=${Math.max(0,current.revision-1)}&to=${current.revision}`));});
$('query-form').onsubmit = event => {event.preventDefault();guard(async () => inspect(await client().query({kind: $('kind').value, entityId: $('entity').value, to: $('to').value, filter: filter(), limit: 200, maxDepth: 8}, current.revision)));};
guard(refresh);
