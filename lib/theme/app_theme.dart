import 'package:flutter/material.dart';

class SyncyColors {
  const SyncyColors._();

  static const Color background = Color(0xFF0B0D12);
  static const Color surface = Color(0xFF12151C);
  static const Color surfaceRaised = Color(0xFF171B24);
  static const Color border = Color(0xFF222735);
  static const Color text = Color(0xFFE7E9EE);
  static const Color muted = Color(0xFF8B93A7);
  static const Color accent = Color(0xFF5B8CFF);
  static const Color accentAlt = Color(0xFF7C5CFF);

  static const Color synced = Color(0xFF35C46A);
  static const Color syncing = Color(0xFF5B8CFF);
  static const Color pending = Color(0xFFF2A94E);
  static const Color danger = Color(0xFFF2555A);
  static const Color offline = Color(0xFF6B7280);

  static const Color accentSoft = Color.fromARGB(38, 91, 140, 255);
  static const Color accentGlow = Color.fromARGB(77, 91, 140, 255);
  static const Color brandGlow = Color.fromARGB(82, 124, 92, 255);
  static const Color dangerSoft = Color.fromARGB(38, 242, 85, 90);

  static const LinearGradient brandGradient = LinearGradient(
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
    colors: [accent, accentAlt],
  );
}

class SyncyText {
  const SyncyText._();

  static const List<String> _monoStack = ['Roboto Mono', 'monospace'];

  static const TextStyle screenTitle = TextStyle(
    color: SyncyColors.text,
    fontSize: 24,
    fontWeight: FontWeight.w700,
    letterSpacing: -0.5,
    height: 1.1,
  );
  static const TextStyle dialogTitle = TextStyle(
    color: SyncyColors.text,
    fontSize: 18,
    fontWeight: FontWeight.w700,
  );
  static const TextStyle cardTitle = TextStyle(
    color: SyncyColors.text,
    fontSize: 15,
    fontWeight: FontWeight.w600,
  );
  static const TextStyle eyebrow = TextStyle(
    color: SyncyColors.muted,
    fontSize: 12,
    fontWeight: FontWeight.w700,
    letterSpacing: 1,
  );
  static const TextStyle body = TextStyle(
    color: SyncyColors.text,
    fontSize: 14,
    height: 1.45,
  );
  static const TextStyle muted = TextStyle(
    color: SyncyColors.muted,
    fontSize: 13,
    height: 1.45,
  );
  static const TextStyle stat = TextStyle(
    color: SyncyColors.text,
    fontSize: 22,
    fontWeight: FontWeight.w700,
    letterSpacing: -0.5,
  );
  static const TextStyle mono = TextStyle(
    fontFamily: 'monospace',
    fontFamilyFallback: _monoStack,
    color: SyncyColors.muted,
    fontSize: 13,
    letterSpacing: 0.2,
    height: 1.35,
  );
  static const TextStyle monoStrong = TextStyle(
    fontFamily: 'monospace',
    fontFamilyFallback: _monoStack,
    color: SyncyColors.text,
    fontSize: 13,
    letterSpacing: 0.2,
    height: 1.35,
  );
}

class AppTheme {
  const AppTheme._();

  static ThemeData get dark {
    const scheme = ColorScheme.dark(
      primary: SyncyColors.accent,
      onPrimary: Colors.white,
      secondary: SyncyColors.accentAlt,
      onSecondary: Colors.white,
      surface: SyncyColors.surface,
      onSurface: SyncyColors.text,
      error: SyncyColors.danger,
      onError: Colors.white,
    );

    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.dark,
      colorScheme: scheme,
      scaffoldBackgroundColor: SyncyColors.background,
      splashColor: SyncyColors.accentSoft,
      highlightColor: Colors.transparent,
      dividerTheme: const DividerThemeData(
        color: SyncyColors.border,
        thickness: 1,
        space: 1,
      ),
      textTheme: const TextTheme(
        titleLarge: SyncyText.dialogTitle,
        titleMedium: SyncyText.cardTitle,
        bodyMedium: SyncyText.body,
        bodySmall: SyncyText.muted,
        labelLarge: TextStyle(
          color: SyncyColors.text,
          fontSize: 14,
          fontWeight: FontWeight.w600,
        ),
      ),
      navigationBarTheme: NavigationBarThemeData(
        height: 70,
        backgroundColor: SyncyColors.surface,
        surfaceTintColor: Colors.transparent,
        indicatorColor: SyncyColors.accentSoft,
        indicatorShape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(14),
        ),
        labelBehavior: NavigationDestinationLabelBehavior.alwaysShow,
        iconTheme: WidgetStateProperty.resolveWith((states) {
          final selected = states.contains(WidgetState.selected);
          return IconThemeData(
            size: 24,
            color: selected ? SyncyColors.accent : SyncyColors.muted,
          );
        }),
        labelTextStyle: WidgetStateProperty.resolveWith((states) {
          final selected = states.contains(WidgetState.selected);
          return TextStyle(
            fontSize: 12,
            fontWeight: FontWeight.w600,
            color: selected ? SyncyColors.text : SyncyColors.muted,
          );
        }),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: SyncyColors.surfaceRaised,
        hintStyle: const TextStyle(color: SyncyColors.muted),
        labelStyle: const TextStyle(color: SyncyColors.muted),
        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(14),
          borderSide: const BorderSide(color: SyncyColors.border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(14),
          borderSide: const BorderSide(color: SyncyColors.border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(14),
          borderSide: const BorderSide(color: SyncyColors.accent, width: 1.5),
        ),
        errorBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(14),
          borderSide: const BorderSide(color: SyncyColors.danger),
        ),
        focusedErrorBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(14),
          borderSide: const BorderSide(color: SyncyColors.danger, width: 1.5),
        ),
      ),
      snackBarTheme: SnackBarThemeData(
        backgroundColor: SyncyColors.surfaceRaised,
        contentTextStyle: const TextStyle(color: SyncyColors.text),
        behavior: SnackBarBehavior.floating,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
        ),
      ),
    );
  }
}
