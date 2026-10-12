/** Render the Go projection without deriving facts or paths in the browser. */
export function forest(boxes) {
  const nodes = new Map(boxes.map(box => [box.entity.id, {...box, children: []}]));
  const roots = [];
  for (const box of nodes.values()) {
    const parent = nodes.get(box.parent);
    if (parent) parent.children.push(box); else roots.push(box);
  }
  return roots;
}

export const relationshipColours = {calls:'#f07c7c', writes:'#f3b562', reads:'#70b9ee', depends_on:'#b795ed', publishes:'#72cda6', subscribes:'#e69bc9', implements:'#a7c86a'};
export function connection(a, b, lane = 0, self = false) {
  const centre = r => ({x:r.x+r.width/2,y:r.y+r.height/2});
  const ac=centre(a),bc=centre(b),dx=bc.x-ac.x,dy=bc.y-ac.y, length=Math.hypot(dx,dy)||1;
  const boundary=(r,c,vx,vy)=>{const t=1/Math.max(Math.abs(vx)/(r.width/2||1),Math.abs(vy)/(r.height/2||1),1e-6);return {x:c.x+vx*t,y:c.y+vy*t};};
  const start=self?{x:a.x+a.width-24,y:a.y}:boundary(a,ac,dx,dy);
  const end=self?{x:a.x+a.width,y:a.y+24}:boundary(b,bc,-dx,-dy);
  const control=self?{x:a.x+a.width+32,y:a.y-32-lane*24}:{x:(start.x+end.x)/2-dy/length*lane*48,y:(start.y+end.y)/2+dx/length*lane*48};
  const dot={x:(start.x+2*control.x+end.x)/4,y:(start.y+2*control.y+end.y)/4};
  const angle=Math.atan2(end.y-control.y,end.x-control.x);
  return {start,end,control,dot,path:`M ${start.x} ${start.y} Q ${control.x} ${control.y} ${end.x} ${end.y}`,
    arrow:`${end.x},${end.y} ${end.x-10*Math.cos(angle-.4)},${end.y-10*Math.sin(angle-.4)} ${end.x-10*Math.cos(angle+.4)},${end.y-10*Math.sin(angle+.4)}`};
}

export class BoxView extends HTMLElement {
  disconnectedCallback() { this._observer?.disconnect(); cancelAnimationFrame(this._frame); }
  connectedCallback() { if (this._projection) this.render(); }
  set projection(value) { this._projection = value; this.render(); }
  get projection() { return this._projection; }
  render() {
    this._observer?.disconnect();
    cancelAnimationFrame(this._frame);
    this._hoverBox = null; this._focusBox = null; this._hoverEdge = null; this._focusEdge = null;
    this.replaceChildren();
    if (!this._projection) return;
    if (!this._projection.edges.some(e=>e.id===this._pinned)) this._pinned=null;
    const boxes = new Map(), arrows = new Map(), dots = new Map();
    const activate=()=>{
      const endpoints = new Set();
      for(const edge of this._projection.edges){const active=edge.id===this._pinned||edge.id===this._hoverEdge||edge.id===this._focusEdge||edge.from===this._hoverBox||edge.to===this._hoverBox||edge.from===this._focusBox||edge.to===this._focusBox;
        arrows.get(edge.id)?.classList.toggle('active',active);for(const dot of dots.get(edge.id)||[])dot.setAttribute('aria-pressed',String(edge.id===this._pinned));
        if(active){endpoints.add(edge.from);endpoints.add(edge.to);}}
      for(const [id,element] of boxes)element.classList.toggle('connected',endpoints.has(id));
    };
    const selectEdge=edge=>{this._pinned=this._pinned===edge.id?null:edge.id;activate();emit('select-edge',edge);};
    const schedule=()=>{cancelAnimationFrame(this._frame);this._frame=requestAnimationFrame(()=>positionEdges());};
    const layout = {...{maxBoxesPerRow: 3, minBoxWidth: 220}, ...this._projection.view?.layout};
    this._observer = new ResizeObserver(entries => {
      for (const {target, contentRect} of entries) {
        if (!target.classList.contains('box-grid')) continue;
        const slots = [...target.children].reduce((n, box) => n + Number(box.dataset.span || 1), 0);
        const columns = Math.max(1, Math.min(layout.maxBoxesPerRow, slots, Math.floor((contentRect.width + 12) / (layout.minBoxWidth + 12))));
        target.style.setProperty('--columns', columns);
        for (const box of target.children) box.style.gridColumn = `span ${Math.min(columns, Number(box.dataset.span || 1))}`;
      }
      schedule();
    });
    const grid = () => { const element = document.createElement('div'); element.className = 'box-grid'; this._observer.observe(element); return element; };
    const emit = (name, detail) => this.dispatchEvent(new CustomEvent(name, {detail, bubbles: true}));
    const draw = box => {
      const section = document.createElement('section');
      section.className = 'box';
      section.dataset.entityId = box.entity.id;
      boxes.set(box.entity.id,section);
      section.dataset.span = box.style?.span || 1;
      if (box.style?.colour) section.style.borderLeft = `4px solid ${box.style.colour}`;
      section.setAttribute('aria-label', box.entity.label);
      const head = document.createElement('div'); head.className = 'box-head';
      const toggle = document.createElement('button');
      toggle.textContent = box.collapsed ? '+' : '−';
      toggle.setAttribute('aria-label', `${box.collapsed ? 'Expand' : 'Collapse'} ${box.entity.label}`);
      toggle.setAttribute('aria-expanded', String(!box.collapsed));
      toggle.addEventListener('click', () => emit('toggle-box', box.entity.id));
      const select = document.createElement('button'); select.className = 'box-title';
      select.textContent = `${box.entity.label} · ${box.entity.kind}`;
      select.addEventListener('click', () => emit('select-entity', box.entity.id));
      head.append(select, toggle); section.append(head);
      if (box.entity.description) {const description = document.createElement('p'); description.className = 'box-description'; description.textContent = box.entity.description; section.append(description);}
      const children = grid(); children.classList.add('box-children');
      for (const child of box.children) children.append(draw(child));
      section.append(children); return section;
    };
    const diagram = grid(); diagram.classList.add('diagram');
    for (const root of forest(this._projection.boxes)) diagram.append(draw(root));
    const stage=document.createElement('div');stage.className='graph-stage';stage.append(diagram);
    const svg=document.createElementNS('http://www.w3.org/2000/svg','svg');svg.classList.add('relationship-arrows');svg.setAttribute('aria-hidden','true');stage.append(svg);
    this.append(stage);
    const svgElement=(tag,attrs)=>{const el=document.createElementNS(svg.namespaceURI,tag);for(const [k,v] of Object.entries(attrs))el.setAttribute(k,v);return el;};
    const groups=new Map();
    for(const edge of this._projection.edges){const key=JSON.stringify([edge.from,edge.to].sort());if(!groups.has(key))groups.set(key,[]);groups.get(key).push(edge);}
    const positionEdges=()=>{
      const bounds=stage.getBoundingClientRect();svg.setAttribute('viewBox',`0 0 ${bounds.width} ${bounds.height}`);
      const rect=dot=>{const r=dot?.querySelector('.dot-core').getBoundingClientRect();return r?{x:r.x-bounds.x,y:r.y-bounds.y,width:r.width,height:r.height}:null;};
      for(const group of groups.values())for(let i=0;i<group.length;i++){
        const edge=group[i],endpoints=dots.get(edge.id)||[],a=rect(endpoints[0]),b=rect(endpoints[1]||endpoints[0]);if(!a||!b)continue;
        // Reverse edges share the same physical lane orientation.
        const lane=(i-(group.length-1)/2)*(edge.from<=edge.to?1:-1);
        const route=connection(a,b,lane,edge.from===edge.to),arrow=arrows.get(edge.id);
        const {x,y}=route.dot;
        arrow.querySelector('path').setAttribute('d',`M ${route.start.x} ${route.start.y} Q ${route.control.x} ${route.control.y} ${route.end.x} ${route.end.y}`);
        const angle=Math.atan2(route.end.y-route.control.y,route.end.x-route.control.x);
        arrow.querySelector('polygon').setAttribute('points',`${route.end.x},${route.end.y} ${route.end.x-10*Math.cos(angle-.4)},${route.end.y-10*Math.sin(angle-.4)} ${route.end.x-10*Math.cos(angle+.4)},${route.end.y-10*Math.sin(angle+.4)}`);
        const label=arrow.querySelector('text');label.setAttribute('x',x);label.setAttribute('y',Math.max(16,y-19));
      }
    };
    for(const edge of this._projection.edges){
      const colour=relationshipColours[edge.predicate]||'#a8b7cd';
      const arrow=svgElement('g',{'class':'edge-arrow','data-edge-id':edge.id});
      arrow.append(svgElement('path',{fill:'none',stroke:colour,'stroke-width':2}),svgElement('polygon',{fill:colour}),svgElement('text',{'text-anchor':'middle',fill:colour}));
      arrow.querySelector('text').textContent=`${edge.predicate} · ${edge.members.length}`;svg.append(arrow);arrows.set(edge.id,arrow);
      const endpoints=[];
      for(const entityId of new Set([edge.from,edge.to])){
      const box=boxes.get(entityId);if(!box)continue;
      let row=box.querySelector(':scope>.box-connections');if(!row){row=document.createElement('div');row.className='box-connections';box.append(row);}
      const dot=document.createElement('button');dot.className='relationship-dot';dot.dataset.edgeId=edge.id;dot.dataset.endpointId=entityId;dot.style.setProperty('--dot-colour',colour);
      const label=`${this._projection.boxes.find(b=>b.entity.id===edge.from)?.entity.label} → ${edge.predicate} · ${edge.members.length} assertions → ${this._projection.boxes.find(b=>b.entity.id===edge.to)?.entity.label}`;
      dot.setAttribute('aria-label',`Reveal relationship: ${label}`);dot.title=label;
      dot.setAttribute('aria-description',`Connection marker in ${box.getAttribute('aria-label')}`);
      dot.innerHTML='<span class="dot-core"></span>';
      if(edge.members.length>1){const count=document.createElement('span');count.className='dot-count';count.textContent=edge.members.length;dot.append(count);}
      dot.addEventListener('mouseenter',()=>{this._hoverEdge=edge.id;this._hoverBox=null;activate();});dot.addEventListener('mouseleave',()=>{this._hoverEdge=null;activate();});
      dot.addEventListener('focus',()=>{this._focusEdge=edge.id;this._focusBox=null;activate();});dot.addEventListener('blur',()=>{this._focusEdge=null;activate();});
      dot.addEventListener('click',()=>selectEdge(edge));row.append(dot);endpoints.push(dot);
      }
      dots.set(edge.id,endpoints);
    }
    diagram.addEventListener('pointerover',event=>{this._hoverBox=event.target.closest?.('.relationship-dot')?null:event.target.closest?.('.box')?.dataset.entityId||null;activate();});
    diagram.addEventListener('focusin',event=>{this._focusBox=event.target.closest?.('.relationship-dot')?null:event.target.closest?.('.box')?.dataset.entityId||null;activate();});
    diagram.addEventListener('pointerleave',()=>{this._hoverBox=null;activate();});diagram.addEventListener('focusout',event=>{this._focusBox=event.relatedTarget?.closest?.('.box')?.dataset.entityId||null;activate();});
    stage.addEventListener('keydown',event=>{if(event.key==='Escape'){this._pinned=null;activate();}});
    this._observer.observe(stage);activate();schedule();
    const title = document.createElement('h2'); title.textContent = 'Relationships'; this.append(title);
    const labels = new Map(this._projection.boxes.map(b => [b.entity.id, b.entity.label]));
    const edges = document.createElement('div'); edges.className = 'edges';
    for (const edge of this._projection.edges) {
      const button = document.createElement('button');
      button.textContent = `${labels.get(edge.from)} → ${edge.predicate} · ${edge.members.length} assertion${edge.members.length === 1 ? '' : 's'} → ${labels.get(edge.to)}`;
      button.addEventListener('click', () => selectEdge(edge)); edges.append(button);
    }
    if (!this._projection.edges.length) edges.textContent = 'No relationships in this view. Knowledge completeness is unknown.';
    this.append(edges);
    const omitted = document.createElement('p'); omitted.className = 'muted';
    omitted.textContent = `${this._projection.omittedEntities} entities hidden or omitted; ${this._projection.omittedAssertions} relationships hidden or omitted. Paths must be queried on the server.`;
    this.append(omitted);
  }
}
customElements.define('box-view', BoxView);

export class BoxClient {
  constructor(base = '', token = '') { this.base = base.replace(/\/$/, ''); this.token = token; }
  async request(path, body) {
    const response = await fetch(`${this.base}/api/${path}`, {
      method: body === undefined ? 'GET' : 'POST',
      headers: {'Content-Type': 'application/json', ...(this.token ? {Authorization: `Bearer ${this.token}`} : {})},
      ...(body === undefined ? {} : {body: JSON.stringify(body)})
    });
    const text = await response.text();
    let result;
    try { result = JSON.parse(text); } catch { throw new Error(`HTTP ${response.status}: ${text}`); }
    if (!response.ok) throw new Error(result.error || `HTTP ${response.status}`);
    return result;
  }
  snapshot(revision) { return this.request(`snapshot${revision === undefined ? '' : `?revision=${revision}`}`); }
  view(spec, revision) { return this.request(`view${revision === undefined ? '' : `?revision=${revision}`}`, spec); }
  query(query, revision) { return this.request(`query${revision === undefined ? '' : `?revision=${revision}`}`, query); }
  explain(ids, revision) { return this.request(`explain${revision === undefined ? '' : `?revision=${revision}`}`, {ids}); }
  apply(patch) { return this.request('apply', patch); }
}
