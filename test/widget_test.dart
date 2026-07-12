import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:syncy/app.dart';

void main() {
  testWidgets('Boots to the pairing screen when unpaired', (tester) async {
    SharedPreferences.setMockInitialValues({});
    await tester.pumpWidget(const SyncyApp());
    await tester.pumpAndSettle();
    expect(find.text('Pair with your desktop'), findsOneWidget);
  });
}
