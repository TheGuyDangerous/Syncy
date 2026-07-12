import 'package:flutter/material.dart';

import '../widgets/empty_state.dart';
import '../widgets/page_header.dart';

class HistoryScreen extends StatelessWidget {
  const HistoryScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const SafeArea(
      bottom: false,
      child: Column(
        children: [
          PageHeader(
            title: 'History',
            subtitle: 'Sync activity, as it happens',
          ),
          Expanded(
            child: EmptyState(
              icon: Icons.timeline_rounded,
              title: 'No activity yet',
              message:
                  'When your devices start syncing, every change lands here — files added, updated, and conflicts resolved.',
            ),
          ),
        ],
      ),
    );
  }
}
