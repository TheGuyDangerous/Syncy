import 'package:flutter/material.dart';

enum SyncStatus {
  synced(Color(0xFF35C46A), Color.fromARGB(40, 53, 196, 106), 'Synced'),
  syncing(Color(0xFF5B8CFF), Color.fromARGB(40, 91, 140, 255), 'Syncing'),
  pending(Color(0xFFF2A94E), Color.fromARGB(40, 242, 169, 78), 'Pending'),
  error(Color(0xFFF2555A), Color.fromARGB(40, 242, 85, 90), 'Error'),
  paused(Color(0xFF6B7280), Color.fromARGB(40, 107, 114, 128), 'Paused'),
  offline(Color(0xFF6B7280), Color.fromARGB(40, 107, 114, 128), 'Offline');

  const SyncStatus(this.color, this.glow, this.label);

  final Color color;
  final Color glow;
  final String label;
}

class StatusDot extends StatelessWidget {
  const StatusDot(this.status, {super.key, this.size = 9});

  final SyncStatus status;
  final double size;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        color: status.color,
        shape: BoxShape.circle,
        boxShadow: [
          BoxShadow(
            color: status.glow,
            blurRadius: size * 1.2,
            spreadRadius: size * 0.25,
          ),
        ],
      ),
    );
  }
}

class StatusPill extends StatelessWidget {
  const StatusPill(this.status, {super.key, this.label});

  final SyncStatus status;
  final String? label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: status.glow,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          StatusDot(status, size: 6),
          const SizedBox(width: 7),
          Text(
            label ?? status.label,
            style: TextStyle(
              color: status.color,
              fontSize: 12,
              fontWeight: FontWeight.w600,
              letterSpacing: 0.2,
            ),
          ),
        ],
      ),
    );
  }
}
