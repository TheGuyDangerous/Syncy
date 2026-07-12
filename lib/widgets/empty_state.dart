import 'package:flutter/material.dart';

import '../theme/app_theme.dart';

class EmptyState extends StatelessWidget {
  const EmptyState({
    super.key,
    required this.icon,
    required this.title,
    required this.message,
    this.action,
  });

  final IconData icon;
  final String title;
  final String message;
  final Widget? action;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 68,
              height: 68,
              alignment: Alignment.center,
              decoration: BoxDecoration(
                color: SyncyColors.surfaceRaised,
                borderRadius: BorderRadius.circular(20),
                border: Border.all(color: SyncyColors.border),
              ),
              child: Icon(icon, color: SyncyColors.accent, size: 28),
            ),
            const SizedBox(height: 22),
            Text(title, style: SyncyText.screenTitle, textAlign: TextAlign.center),
            const SizedBox(height: 10),
            Text(message, style: SyncyText.muted, textAlign: TextAlign.center),
            if (action != null) ...[
              const SizedBox(height: 24),
              action!,
            ],
          ],
        ),
      ),
    );
  }
}
