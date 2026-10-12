import 'package:box_view_example/main.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('viewer starts and offers API editing', (tester) async {
    await tester.pumpWidget(const BoxViewApp());
    expect(find.text('Box View'), findsOneWidget);
    expect(find.text('Apply an atomic patch'), findsOneWidget);
    await tester.pump(const Duration(seconds: 1));
  });
}
