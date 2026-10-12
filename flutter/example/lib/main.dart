import 'dart:convert';
import 'package:box_view/box_view.dart';
import 'package:flutter/material.dart';

void main() => runApp(const BoxViewApp());

class BoxViewApp extends StatelessWidget {
  const BoxViewApp({super.key});
  @override
  Widget build(BuildContext context) => MaterialApp(
    title: 'Box View',
    theme: ThemeData.dark(useMaterial3: true),
    home: const Viewer(),
  );
}

class Viewer extends StatefulWidget {
  const Viewer({super.key});
  @override
  State<Viewer> createState() => _ViewerState();
}

class _ViewerState extends State<Viewer> {
  final client = BoxClient(
    const String.fromEnvironment(
      'BOX_VIEW_URL',
      defaultValue: 'http://127.0.0.1:8080',
    ),
    token: const String.fromEnvironment('BOX_VIEW_TOKEN'),
  );
  final collapsed = <String>{};
  final patch = TextEditingController();
  Projection? projection;
  String inspector = 'Select a component or relationship.',
      status = 'Loading…',
      lens = '';
  int generation = 0;
  int maxBoxesPerRow = 0;
  Json get filter => {
    'hierarchy': 'architecture',
    'dispositions': ['accepted'],
    'applicabilities': ['applicable'],
    'predicates': lens.isEmpty ? <String>[] : [lens],
  };
  @override
  void initState() {
    super.initState();
    refresh();
  }

  @override
  void dispose() {
    client.close();
    patch.dispose();
    super.dispose();
  }

  Future<void> guard(Future<void> Function() work) async {
    try {
      await work();
    } catch (error) {
      if (mounted) setState(() => status = '$error');
    }
  }

  Future<void> refresh() => guard(() async {
    final request = ++generation;
    final result = await client.view({
      'filter': filter,
      'collapsed': collapsed.toList(),
      'layout': {'maxBoxesPerRow': maxBoxesPerRow},
    });
    if (!mounted || request != generation) return;
    setState(() {
      projection = result;
      status = 'Revision ${result.revision}';
    });
    if (patch.text.isEmpty) {
      patch.text = const JsonEncoder.withIndent('  ').convert({
        'expectedRevision': result.revision,
        'actor': 'human',
        'reason': 'Rename Sync',
        'operations': [
          {
            'action': 'update',
            'entity': {
              'id': 'sync',
              'kind': 'component',
              'label': 'Synchronizer',
            },
          },
        ],
      });
    }
  });
  Future<void> show(Future<Json> result) => guard(() async {
    final value = await result;
    if (mounted) {
      setState(
        () => inspector = const JsonEncoder.withIndent('  ').convert(value),
      );
    }
  });
  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      title: const Text('Box View'),
      actions: [
        IconButton(
          onPressed: refresh,
          icon: const Icon(Icons.refresh),
          tooltip: 'Refresh',
        ),
      ],
    ),
    body: ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text(status),
        DropdownButton<int>(
          value: maxBoxesPerRow,
          items: [0, 1, 2, 3, 4, 6]
              .map(
                (n) => DropdownMenuItem(
                  value: n,
                  child: Text(
                    n == 0 ? 'Server row default' : 'Max $n boxes per row',
                  ),
                ),
              )
              .toList(),
          onChanged: (n) {
            setState(() => maxBoxesPerRow = n!);
            refresh();
          },
        ),
        DropdownButton<String>(
          value: lens,
          items: ['', 'calls', 'writes', 'reads', 'depends_on']
              .map(
                (v) => DropdownMenuItem(
                  value: v,
                  child: Text(v.isEmpty ? 'All relationships' : v),
                ),
              )
              .toList(),
          onChanged: (v) {
            setState(() => lens = v!);
            refresh();
          },
        ),
        if (projection != null)
          BoxDiagram(
            projection: projection!,
            onToggle: (id) {
              if (!collapsed.add(id)) collapsed.remove(id);
              refresh();
            },
            onSelectEdge: (edge) => show(
              client.explain(edge.members, revision: projection!.revision),
            ),
            onSelectEntity: (id) => show(
              client.query({
                'kind': 'boundary',
                'entityId': id,
                'filter': filter,
              }, revision: projection!.revision),
            ),
          ),
        const Divider(),
        const Text('Inspector'),
        SelectableText(inspector),
        ExpansionTile(
          title: const Text('Apply an atomic patch'),
          children: [
            TextField(
              controller: patch,
              minLines: 5,
              maxLines: 14,
              decoration: const InputDecoration(labelText: 'JSON patch'),
            ),
            FilledButton(
              onPressed: () => guard(() async {
                await show(client.apply(jsonDecode(patch.text) as Json));
                await refresh();
              }),
              child: const Text('Apply patch'),
            ),
          ],
        ),
      ],
    ),
  );
}
