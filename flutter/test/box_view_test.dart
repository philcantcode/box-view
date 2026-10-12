import 'dart:convert';
import 'package:box_view/box_view.dart';
import 'package:flutter/material.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

void main() {
  testWidgets(
    'dots reveal arrows, pin exact members and reposition on resize',
    (tester) async {
      final p = Projection.fromJson({
        'revision': 1,
        'omittedEntities': 0,
        'omittedAssertions': 0,
        'boxes': [
          {
            'entity': {'id': 'root', 'label': 'Root', 'kind': 'system'},
            'collapsed': false,
          },
          {
            'entity': {'id': 'a', 'label': 'A', 'kind': 'component'},
            'parent': 'root',
            'collapsed': false,
          },
          {
            'entity': {'id': 'b', 'label': 'B', 'kind': 'component'},
            'parent': 'root',
            'collapsed': false,
          },
        ],
        'edges': [
          {
            'id': 'ab',
            'from': 'a',
            'to': 'b',
            'predicate': 'calls',
            'members': ['one', 'two'],
          },
        ],
      });
      List<String>? selected;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SingleChildScrollView(
              child: BoxDiagram(
                projection: p,
                onSelectEntity: (_) {},
                onSelectEdge: (edge) => selected = edge.members,
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      dynamic painter() => tester
          .widget<CustomPaint>(
            find.byWidgetPredicate(
              (w) =>
                  w is CustomPaint &&
                  w.painter.runtimeType.toString() == '_RelationshipPainter',
            ),
          )
          .painter;
      final dot = find.byKey(const ValueKey('relationship-dot-ab-a'));
      final otherDot = find.byKey(const ValueKey('relationship-dot-ab-b'));
      expect(otherDot, findsOneWidget);
      for (final id in ['a', 'b']) {
        final box = find.byKey(ValueKey('box-$id')),
            marker = find.byKey(ValueKey('relationship-dot-ab-$id'));
        expect(
          tester.getTopLeft(marker).dx - tester.getTopLeft(box).dx,
          lessThan(25),
        );
        expect(
          tester.getBottomLeft(box).dy - tester.getBottomLeft(marker).dy,
          lessThan(25),
        );
      }
      expect(dot, findsOneWidget);
      expect(painter().active, isEmpty);
      final mouse = await tester.createGesture(kind: PointerDeviceKind.mouse);
      await mouse.addPointer(location: Offset.zero);
      await mouse.moveTo(tester.getCenter(find.text('A · component')));
      await tester.pumpAndSettle();
      expect(painter().active, contains('ab'));
      await mouse.moveTo(const Offset(1, 700));
      await tester.pumpAndSettle();
      expect(painter().active, isEmpty);
      Focus.of(tester.element(find.text('A · component'))).requestFocus();
      await tester.pumpAndSettle();
      expect(painter().active, contains('ab'));
      FocusManager.instance.primaryFocus?.unfocus();
      await tester.pumpAndSettle();
      await tester.tap(dot);
      await tester.pumpAndSettle();
      expect(selected, ['one', 'two']);
      expect(painter().active, contains('ab'));
      Semantics markerSemantics(Finder marker) => tester.widget<Semantics>(
        find.ancestor(of: marker, matching: find.byType(Semantics)).first,
      );
      expect(markerSemantics(dot).properties.toggled, true);
      expect(markerSemantics(otherDot).properties.toggled, true);
      await tester.tap(otherDot);
      await tester.pumpAndSettle();
      expect(markerSemantics(dot).properties.toggled, false);
      expect(markerSemantics(otherDot).properties.toggled, false);
      expect(selected, ['one', 'two']);
      await tester.tap(dot);
      await tester.pumpAndSettle();
      await tester.sendKeyEvent(LogicalKeyboardKey.escape);
      FocusManager.instance.primaryFocus?.unfocus();
      await tester.pumpAndSettle();
      expect(painter().active, isEmpty);
      final before = tester.getTopLeft(otherDot);
      tester.view.physicalSize = const Size(320, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpAndSettle();
      expect(tester.getTopLeft(otherDot), isNot(before));
      expect(tester.takeException(), isNull);
      await mouse.removePointer();
    },
  );
  testWidgets('descriptions, colours, spans and responsive row limits', (
    tester,
  ) async {
    final projection = Projection.fromJson({
      'revision': 1,
      'omittedEntities': 0,
      'omittedAssertions': 0,
      'edges': [],
      'view': {
        'layout': {'maxBoxesPerRow': 3, 'minBoxWidth': 220},
      },
      'boxes': [
        {
          'entity': {
            'id': 'a',
            'label': 'A',
            'kind': 'component',
            'description': 'A description',
          },
          'collapsed': false,
          'style': {'colour': '#4f83cc', 'span': 2},
        },
        {
          'entity': {'id': 'b', 'label': 'B', 'kind': 'component'},
          'collapsed': false,
          'style': {'span': 1},
        },
      ],
    });
    Widget widget() => MaterialApp(
      home: Scaffold(
        body: SingleChildScrollView(child: BoxDiagram(projection: projection)),
      ),
    );
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(900, 900);
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.pumpWidget(widget());
    await tester.pumpAndSettle();
    final a = find.byKey(const ValueKey('box-a')),
        b = find.byKey(const ValueKey('box-b'));
    expect(find.text('A description'), findsOneWidget);
    expect(tester.getTopLeft(a).dy, tester.getTopLeft(b).dy);
    expect(
      tester.getSize(a).width,
      closeTo(tester.getSize(b).width * 2 + 12, 0.1),
    );
    final card = tester.widget<Card>(
      find.descendant(of: a, matching: find.byType(Card)).first,
    );
    expect(
      (card.shape as RoundedRectangleBorder).side.color,
      const Color(0xff4f83cc),
    );
    tester.view.physicalSize = const Size(320, 900);
    await tester.pumpAndSettle();
    expect(tester.getSize(a).width, 320);
    expect(tester.getTopLeft(b).dy, greaterThan(tester.getTopLeft(a).dy));
    expect(tester.takeException(), isNull);
  });
  test('client pins explanations and preserves revision conflicts', () async {
    final client = BoxClient(
      'http://localhost:8080',
      token: 'token',
      client: MockClient((request) async {
        expect(request.url.path, '/api/explain');
        expect(request.url.queryParameters['revision'], '12');
        expect(request.headers['Authorization'], 'Bearer token');
        expect(jsonDecode(request.body), {
          'ids': ['a'],
        });
        return http.Response('{"revision":12}', 200);
      }),
    );
    expect((await client.explain(['a'], revision: 12))['revision'], 12);
    client.close();
    final conflict = BoxClient(
      'http://localhost',
      client: MockClient(
        (_) async => http.Response('{"error":"revision conflict"}', 409),
      ),
    );
    await expectLater(
      conflict.apply({}),
      throwsA(isA<BoxApiException>().having((e) => e.status, 'status', 409)),
    );
    conflict.close();
  });
  testWidgets('nested boxes and aggregates retain IDs for interaction', (
    tester,
  ) async {
    final projection = Projection.fromJson({
      'revision': 2,
      'omittedEntities': 1,
      'omittedAssertions': 0,
      'boxes': [
        {
          'entity': {'id': 'storage', 'label': 'Storage', 'kind': 'component'},
          'collapsed': true,
        },
        {
          'entity': {'id': 'sync', 'label': 'Sync', 'kind': 'component'},
          'collapsed': false,
        },
      ],
      'edges': [
        {
          'id': 'group',
          'from': 'sync',
          'to': 'storage',
          'predicate': 'writes',
          'members': ['a', 'b'],
        },
      ],
    });
    String? toggle;
    List<String>? members;
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: BoxDiagram(
              projection: projection,
              onToggle: (id) => toggle = id,
              onSelectEdge: (edge) => members = edge.members,
            ),
          ),
        ),
      ),
    );
    expect(find.text('Storage · component'), findsOneWidget);
    await tester.tap(find.text('+'));
    expect(toggle, 'storage');
    await tester.tap(find.text('Sync → writes · 2 assertions → Storage'));
    expect(members, ['a', 'b']);
  });
}
