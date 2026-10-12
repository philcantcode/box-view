part of 'box_view.dart';

Color _relationshipColour(String predicate) => Color(
  const {
        'calls': 0xfff07c7c,
        'writes': 0xfff3b562,
        'reads': 0xff70b9ee,
        'depends_on': 0xffb795ed,
        'publishes': 0xff72cda6,
        'subscribes': 0xffe69bc9,
        'implements': 0xffa7c86a,
      }[predicate] ??
      0xffa8b7cd,
);

String _endpoint(ModelEdge edge, String entityId) =>
    jsonEncode([edge.id, entityId]);
typedef _GraphBuilder =
    Widget Function(
      BuildContext,
      GlobalKey Function(String),
      ValueChanged<String?>,
      bool Function(String),
      ValueChanged<ModelEdge>,
      Widget Function(String),
    );

class _RelationshipGraph extends StatefulWidget {
  const _RelationshipGraph({
    required this.projection,
    required this.builder,
    this.onSelectEdge,
  });
  final Projection projection;
  final _GraphBuilder builder;
  final ValueChanged<ModelEdge>? onSelectEdge;
  @override
  State<_RelationshipGraph> createState() => _RelationshipGraphState();
}

class _RelationshipGraphState extends State<_RelationshipGraph> {
  final _stageKey = GlobalKey();
  final _cards = <String, GlobalKey>{}, _dotKeys = <String, GlobalKey>{};
  final _dotFocus = <String, FocusNode>{};
  Map<String, Rect> _dotRects = {}, _cardRects = {};
  String? _hoverBox, _focusBox, _hoverEdge, _focusEdge, _pinned;
  bool _measuring = false;
  bool _active(ModelEdge edge) =>
      edge.id == _pinned ||
      edge.id == _hoverEdge ||
      edge.id == _focusEdge ||
      edge.from == _hoverBox ||
      edge.to == _hoverBox ||
      edge.from == _focusBox ||
      edge.to == _focusBox;
  void _select(ModelEdge edge, [String? entityId]) {
    setState(() => _pinned = _pinned == edge.id ? null : edge.id);
    _dotFocus[_endpoint(edge, entityId ?? edge.from)]?.requestFocus();
    widget.onSelectEdge?.call(edge);
  }

  void _measure() {
    if (_measuring) return;
    _measuring = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _measuring = false;
      if (!mounted) return;
      final stage = _stageKey.currentContext?.findRenderObject();
      if (stage is! RenderBox || !stage.hasSize) return;
      Map<String, Rect> measure(
        Map<String, GlobalKey> keys, {
        bool dots = false,
      }) {
        final rects = <String, Rect>{};
        for (final entry in keys.entries) {
          final box = entry.value.currentContext?.findRenderObject();
          if (box is RenderBox && box.hasSize) {
            final rect =
                box.localToGlobal(Offset.zero, ancestor: stage) & box.size;
            rects[entry.key] = dots ? rect.deflate(8) : rect;
          }
        }
        return rects;
      }

      final dots = measure(_dotKeys, dots: true), cards = measure(_cards);
      if (!mapEquals(dots, _dotRects) || !mapEquals(cards, _cardRects)) {
        setState(() {
          _dotRects = dots;
          _cardRects = cards;
        });
      }
    });
  }

  @override
  void didUpdateWidget(_RelationshipGraph oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!widget.projection.edges.any((e) => e.id == _pinned)) _pinned = null;
    _cards.removeWhere(
      (id, _) => !widget.projection.boxes.any((b) => b.id == id),
    );
    final endpoints = {
      for (final edge in widget.projection.edges)
        for (final id in {edge.from, edge.to}) _endpoint(edge, id),
    };
    for (final key
        in _dotFocus.keys.where((key) => !endpoints.contains(key)).toList()) {
      _dotFocus.remove(key)?.dispose();
    }
    _dotKeys.removeWhere((key, _) => !endpoints.contains(key));
  }

  @override
  void dispose() {
    for (final node in _dotFocus.values) {
      node.dispose();
    }
    super.dispose();
  }

  void _hover(Offset position) {
    final stage = _stageKey.currentContext?.findRenderObject();
    if (stage is! RenderBox) return;
    final local = stage.globalToLocal(position);
    String? id;
    double area = double.infinity;
    if (_hoverEdge == null) {
      for (final entry in _cardRects.entries) {
        if (entry.value.contains(local) &&
            entry.value.width * entry.value.height < area) {
          id = entry.key;
          area = entry.value.width * entry.value.height;
        }
      }
    }
    if (_hoverBox != id) setState(() => _hoverBox = id);
  }

  Widget _dotsFor(String entityId) {
    final edges = widget.projection.edges
        .where((e) => e.from == entityId || e.to == entityId)
        .toList();
    if (edges.isEmpty) return const SizedBox.shrink();
    final labels = {
      for (final box in widget.projection.boxes) box.id: box.label,
    };
    return Padding(
      padding: const EdgeInsets.only(top: 10),
      child: Align(
        alignment: Alignment.bottomLeft,
        child: Wrap(
          spacing: 4,
          runSpacing: 4,
          children: [
            for (final edge in edges)
              SizedBox(
                key: _dotKeys.putIfAbsent(
                  _endpoint(edge, entityId),
                  () => GlobalKey(),
                ),
                width: 32,
                height: 32,
                child: MouseRegion(
                  onEnter: (_) => setState(() {
                    _hoverEdge = edge.id;
                    _hoverBox = null;
                  }),
                  onExit: (_) => setState(() => _hoverEdge = null),
                  child: Focus(
                    onFocusChange: (focused) => setState(
                      () => _focusEdge = focused
                          ? edge.id
                          : (_focusEdge == edge.id ? null : _focusEdge),
                    ),
                    child: Semantics(
                      button: true,
                      toggled: _pinned == edge.id,
                      child: IconButton(
                        key: ValueKey('relationship-dot-${edge.id}-$entityId'),
                        focusNode: _dotFocus.putIfAbsent(
                          _endpoint(edge, entityId),
                          () => FocusNode(),
                        ),
                        tooltip:
                            'Reveal relationship: ${labels[edge.from]} → ${edge.predicate} · ${edge.members.length} assertions → ${labels[edge.to]}',
                        padding: EdgeInsets.zero,
                        constraints: const BoxConstraints.tightFor(
                          width: 32,
                          height: 32,
                        ),
                        onPressed: () => _select(edge, entityId),
                        icon: SizedBox(
                          width: 32,
                          height: 32,
                          child: Stack(
                            alignment: Alignment.center,
                            children: [
                              Container(
                                width: 14,
                                height: 14,
                                decoration: BoxDecoration(
                                  shape: BoxShape.circle,
                                  color: _relationshipColour(edge.predicate),
                                  border: Border.all(
                                    color: const Color(0xff10151e),
                                    width: 2,
                                  ),
                                ),
                              ),
                              if (edge.members.length > 1)
                                Positioned(
                                  top: 0,
                                  right: 0,
                                  child: Text(
                                    '${edge.members.length}',
                                    style: const TextStyle(
                                      fontSize: 10,
                                      backgroundColor: Color(0xff10151e),
                                    ),
                                  ),
                                ),
                            ],
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    _measure();
    final routes = <String, _Connection>{},
        groups = <String, List<ModelEdge>>{};
    for (final edge in widget.projection.edges) {
      final pair = [edge.from, edge.to]..sort();
      (groups[jsonEncode(pair)] ??= []).add(edge);
    }
    for (final group in groups.values) {
      for (var i = 0; i < group.length; i++) {
        final edge = group[i],
            a = _dotRects[_endpoint(edge, edge.from)],
            b = _dotRects[_endpoint(edge, edge.to)];
        if (a == null || b == null) continue;
        routes[edge.id] = _Connection.between(
          a,
          b,
          (i - (group.length - 1) / 2) *
              (edge.from.compareTo(edge.to) <= 0 ? 1 : -1),
          edge.from == edge.to,
        );
      }
    }
    final content = widget.builder(
      context,
      (id) => _cards.putIfAbsent(id, () => GlobalKey()),
      (id) {
        if (_focusBox != id) setState(() => _focusBox = id);
      },
      (id) => widget.projection.edges.any(
        (e) => _active(e) && (e.from == id || e.to == id),
      ),
      (edge) => _select(edge),
      _dotsFor,
    );
    return CallbackShortcuts(
      bindings: {
        const SingleActivator(LogicalKeyboardKey.escape): () {
          setState(() => _pinned = null);
        },
      },
      child: MouseRegion(
        onHover: (event) => _hover(event.position),
        onExit: (_) => setState(() => _hoverBox = null),
        child: Stack(
          key: _stageKey,
          children: [
            NotificationListener<SizeChangedLayoutNotification>(
              onNotification: (_) {
                _measure();
                return true;
              },
              child: SizeChangedLayoutNotifier(child: content),
            ),
            Positioned.fill(
              child: IgnorePointer(
                child: CustomPaint(
                  painter: _RelationshipPainter(
                    widget.projection.edges,
                    routes,
                    widget.projection.edges
                        .where(_active)
                        .map((e) => e.id)
                        .toSet(),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _Connection {
  const _Connection(this.start, this.control, this.end);
  final Offset start, control, end;
  Offset get dot => (start + control * 2 + end) / 4;
  factory _Connection.between(Rect a, Rect b, double lane, bool self) {
    final delta = b.center - a.center,
        length = delta.distance == 0 ? 1.0 : delta.distance;
    Offset boundary(Rect r, Offset direction) {
      final ratio = math.max(
        direction.dx.abs() / math.max(1, r.width / 2),
        direction.dy.abs() / math.max(1, r.height / 2),
      );
      return r.center + direction / math.max(0.000001, ratio);
    }

    final start = self ? Offset(a.right - 24, a.top) : boundary(a, delta),
        end = self ? Offset(a.right, a.top + 24) : boundary(b, -delta);
    final control = self
        ? Offset(a.right + 32, a.top - 32 - lane * 24)
        : (start + end) / 2 +
              Offset(-delta.dy, delta.dx) / length * (lane * 48);
    return _Connection(start, control, end);
  }
}

class _RelationshipPainter extends CustomPainter {
  _RelationshipPainter(this.edges, this.routes, this.active);
  final List<ModelEdge> edges;
  final Map<String, _Connection> routes;
  final Set<String> active;
  @override
  void paint(Canvas canvas, Size size) {
    for (final edge in edges) {
      final r = routes[edge.id];
      if (r == null || !active.contains(edge.id)) continue;
      final colour = _relationshipColour(edge.predicate);
      canvas.drawPath(
        Path()
          ..moveTo(r.start.dx, r.start.dy)
          ..quadraticBezierTo(r.control.dx, r.control.dy, r.end.dx, r.end.dy),
        Paint()
          ..color = colour
          ..strokeWidth = 2
          ..style = PaintingStyle.stroke,
      );
      final angle = math.atan2(
        r.end.dy - r.control.dy,
        r.end.dx - r.control.dx,
      );
      final tip1 =
              r.end - Offset(math.cos(angle - .4), math.sin(angle - .4)) * 10,
          tip2 =
              r.end - Offset(math.cos(angle + .4), math.sin(angle + .4)) * 10;
      canvas.drawPath(
        Path()
          ..moveTo(r.end.dx, r.end.dy)
          ..lineTo(tip1.dx, tip1.dy)
          ..lineTo(tip2.dx, tip2.dy)
          ..close(),
        Paint()..color = colour,
      );
      final text = TextPainter(
        text: TextSpan(
          text: '${edge.predicate} · ${edge.members.length}',
          style: TextStyle(
            color: colour,
            fontSize: 13,
            fontWeight: FontWeight.w600,
          ),
        ),
        textDirection: TextDirection.ltr,
      )..layout(maxWidth: size.width);
      final position = Offset(
        (r.dot.dx - text.width / 2).clamp(
          0,
          math.max(0, size.width - text.width),
        ),
        math.max(0, r.dot.dy - 24),
      );
      canvas.drawRRect(
        RRect.fromRectAndRadius(
          (position & text.size).inflate(3),
          const Radius.circular(3),
        ),
        Paint()..color = const Color(0xff10151e),
      );
      text.paint(canvas, position);
    }
  }

  @override
  bool shouldRepaint(covariant _RelationshipPainter oldDelegate) => true;
}
