library;

import 'dart:convert';
import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart' show mapEquals;
import 'package:flutter/services.dart';
import 'package:http/http.dart' as http;

part 'connections.dart';

typedef Json = Map<String, dynamic>;

/// Shared protocol client. Semantic queries and projection run in Go.
class BoxClient {
  BoxClient(this.baseUrl, {this.token = '', http.Client? client})
    : _client = client ?? http.Client();
  final String baseUrl;
  final String token;
  final http.Client _client;
  Future<dynamic> request(String path, {Json? body}) async {
    final uri = Uri.parse('${baseUrl.replaceAll(RegExp(r'/$'), '')}/api/$path');
    final headers = <String, String>{
      'Content-Type': 'application/json',
      if (token.isNotEmpty) 'Authorization': 'Bearer $token',
    };
    final response = body == null
        ? await _client.get(uri, headers: headers)
        : await _client.post(uri, headers: headers, body: jsonEncode(body));
    final dynamic decoded;
    try {
      decoded = jsonDecode(response.body);
    } on FormatException {
      throw BoxApiException(response.statusCode, response.body);
    }
    if (response.statusCode >= 400) {
      throw BoxApiException(
        response.statusCode,
        decoded is Map ? '${decoded['error']}' : response.body,
      );
    }
    return decoded;
  }

  String _at(String path, int? revision) =>
      revision == null ? path : '$path?revision=$revision';
  Future<Json> snapshot({int? revision}) async =>
      await request(_at('snapshot', revision)) as Json;
  Future<Projection> view(Json spec, {int? revision}) async =>
      Projection.fromJson(
        await request(_at('view', revision), body: spec) as Json,
      );
  Future<Json> query(Json query, {int? revision}) async =>
      await request(_at('query', revision), body: query) as Json;
  Future<Json> explain(List<String> ids, {int? revision}) async =>
      await request(_at('explain', revision), body: {'ids': ids}) as Json;
  Future<Json> apply(Json patch) async =>
      await request('apply', body: patch) as Json;
  void close() => _client.close();
}

class BoxApiException implements Exception {
  const BoxApiException(this.status, this.message);
  final int status;
  final String message;
  @override
  String toString() => 'HTTP $status: $message';
}

class ModelBox {
  ModelBox.fromJson(Json json)
    : id = json['entity']['id'] as String,
      label = json['entity']['label'] as String,
      kind = json['entity']['kind'] as String,
      description = json['entity']['description'] as String? ?? '',
      colour = (json['style'] as Json?)?['colour'] as String? ?? '',
      span = (json['style'] as Json?)?['span'] as int? ?? 1,
      parent = json['parent'] as String? ?? '',
      collapsed = json['collapsed'] as bool;
  final String id, label, kind, parent, description, colour;
  final int span;
  final bool collapsed;
}

class ModelEdge {
  ModelEdge.fromJson(Json json)
    : id = json['id'] as String,
      from = json['from'] as String,
      to = json['to'] as String,
      predicate = json['predicate'] as String,
      members = List<String>.from(json['members'] as List);
  final String id, from, to, predicate;
  final List<String> members;
}

class LayoutConfig {
  const LayoutConfig({this.maxBoxesPerRow = 3, this.minBoxWidth = 220})
    : assert(maxBoxesPerRow >= 1 && maxBoxesPerRow <= 12),
      assert(minBoxWidth >= 120 && minBoxWidth <= 1200);
  factory LayoutConfig.fromJson(Json json) => LayoutConfig(
    maxBoxesPerRow: json['maxBoxesPerRow'] as int? ?? 3,
    minBoxWidth: json['minBoxWidth'] as int? ?? 220,
  );
  final int maxBoxesPerRow, minBoxWidth;
}

class Projection {
  Projection.fromJson(Json json)
    : revision = json['revision'] as int,
      boxes = (json['boxes'] as List)
          .map((b) => ModelBox.fromJson(b as Json))
          .toList(),
      edges = (json['edges'] as List)
          .map((e) => ModelEdge.fromJson(e as Json))
          .toList(),
      omittedEntities = json['omittedEntities'] as int,
      omittedAssertions = json['omittedAssertions'] as int,
      layout = LayoutConfig.fromJson(
        (json['view'] as Json?)?['layout'] as Json? ?? {},
      );
  final LayoutConfig layout;
  final int revision, omittedEntities, omittedAssertions;
  final List<ModelBox> boxes;
  final List<ModelEdge> edges;
}

/// Accessible nested cards and exact relationship groups.
class BoxDiagram extends StatelessWidget {
  const BoxDiagram({
    super.key,
    required this.projection,
    this.onToggle,
    this.onSelectEntity,
    this.onSelectEdge,
    this.layout,
  });
  final Projection projection;
  final LayoutConfig? layout;
  final ValueChanged<String>? onToggle, onSelectEntity;
  final ValueChanged<ModelEdge>? onSelectEdge;
  @override
  Widget build(BuildContext context) {
    return _RelationshipGraph(
      projection: projection,
      onSelectEdge: onSelectEdge,
      builder: (context, cardFor, focus, connected, selectEdge, dotsFor) {
        final children = <String, List<ModelBox>>{};
        final labels = {for (final box in projection.boxes) box.id: box.label};
        for (final box in projection.boxes) {
          (children[labels.containsKey(box.parent) ? box.parent : ''] ??= [])
              .add(box);
        }
        final config = layout ?? projection.layout;
        late Widget Function(ModelBox, int) draw;
        Widget row(List<ModelBox> boxes, int depth) {
          if (boxes.isEmpty) return const SizedBox.shrink();
          return LayoutBuilder(
            builder: (context, constraints) {
              final available = constraints.hasBoundedWidth
                  ? constraints.maxWidth
                  : MediaQuery.sizeOf(context).width;
              final slots = boxes.fold<int>(
                0,
                (n, box) => n + math.max(1, box.span),
              );
              final columns = math.max(
                1,
                math.min(
                  config.maxBoxesPerRow,
                  math.min(
                    slots,
                    ((available + 12) / (config.minBoxWidth + 12)).floor(),
                  ),
                ),
              );
              final unit = math.max(
                0.0,
                (available - 12 * (columns - 1)) / columns,
              );
              return Wrap(
                spacing: 12,
                runSpacing: 12,
                children: [
                  for (final box in boxes)
                    SizedBox(
                      key: ValueKey('box-${box.id}'),
                      width:
                          unit * math.min(columns, math.max(1, box.span)) +
                          12 * (math.min(columns, math.max(1, box.span)) - 1),
                      child: draw(box, depth),
                    ),
                ],
              );
            },
          );
        }

        draw = (ModelBox box, int depth) => Card(
          key: cardFor(box.id),
          margin: EdgeInsets.zero,
          color: Color.lerp(
            const Color(0xff182435),
            const Color(0xff3c506b),
            (depth / 8).clamp(0, 1),
          ),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
            side: BorderSide(
              color: RegExp(r'^#[0-9a-fA-F]{6}$').hasMatch(box.colour)
                  ? Color(
                      0xff000000 |
                          int.parse(box.colour.substring(1), radix: 16),
                    )
                  : const Color(0xff506781),
              width: connected(box.id) ? 3 : (box.colour.isEmpty ? 1 : 3),
            ),
          ),
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Focus(
                  onFocusChange: (value) => focus(value ? box.id : null),
                  child: Wrap(
                    alignment: WrapAlignment.spaceBetween,
                    crossAxisAlignment: WrapCrossAlignment.center,
                    children: [
                      TextButton(
                        onPressed: onSelectEntity == null
                            ? null
                            : () => onSelectEntity!(box.id),
                        child: Text('${box.label} · ${box.kind}'),
                      ),
                      TextButton(
                        onPressed: onToggle == null
                            ? null
                            : () => onToggle!(box.id),
                        child: Semantics(
                          label:
                              '${box.collapsed ? 'Expand' : 'Collapse'} ${box.label}',
                          excludeSemantics: true,
                          child: Text(box.collapsed ? '+' : '−'),
                        ),
                      ),
                    ],
                  ),
                ),
                if (box.description.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 8),
                    child: Text(box.description),
                  ),
                row(children[box.id] ?? <ModelBox>[], depth + 1),
                dotsFor(box.id),
              ],
            ),
          ),
        );
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text('Revision ${projection.revision}'),
            row(children[''] ?? <ModelBox>[], 0),
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 12),
              child: Text('Relationships'),
            ),
            for (final edge in projection.edges)
              OutlinedButton(
                onPressed: () => selectEdge(edge),
                child: Text(
                  '${labels[edge.from]} → ${edge.predicate} · ${edge.members.length} assertions → ${labels[edge.to]}',
                ),
              ),
            if (projection.edges.isEmpty)
              const Text(
                'No relationships in this view. Knowledge completeness is unknown.',
              ),
            Text(
              '${projection.omittedEntities} entities hidden or omitted; ${projection.omittedAssertions} relationships hidden or omitted.',
            ),
          ],
        );
      },
    );
  }
}
