import 'package:flutter/material.dart';

import '../models/conflict.dart';
import '../models/device.dart';
import '../models/device_status.dart';
import '../services/api_client.dart';
import '../theme/app_theme.dart';
import '../widgets/brand_button.dart';
import '../widgets/brand_mark.dart';
import '../widgets/empty_state.dart';
import '../widgets/page_header.dart';
import '../widgets/section_card.dart';
import '../widgets/status_indicators.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key, required this.api});

  final SyncyApi api;

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  bool _loading = true;
  String? _error;
  DeviceStatus? _status;
  List<Device> _devices = const [];
  List<Conflict> _conflicts = const [];

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final status = await widget.api.status();
      final devices = await widget.api.devices();
      List<Conflict> conflicts = const [];
      try {
        conflicts = await widget.api.conflicts();
      } on ApiException {
        conflicts = const [];
      }
      if (!mounted) return;
      setState(() {
        _status = status;
        _devices = devices;
        _conflicts = conflicts;
        _loading = false;
      });
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.message;
        _loading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      bottom: false,
      child: Column(
        children: [
          PageHeader(
            leading: const BrandMark(size: 44),
            title: 'Syncy',
            subtitle: 'Companion for your desktop',
            trailing: RefreshButton(loading: _loading, onTap: _load),
          ),
          Expanded(
            child: RefreshIndicator(
              onRefresh: _load,
              color: SyncyColors.accent,
              backgroundColor: SyncyColors.surfaceRaised,
              child: _content(),
            ),
          ),
        ],
      ),
    );
  }

  Widget _content() {
    if (_loading && _status == null) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: const [
          SizedBox(height: 220),
          Center(child: CircularProgressIndicator()),
        ],
      );
    }
    if (_error != null && _status == null) {
      return ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        children: [
          const SizedBox(height: 40),
          EmptyState(
            icon: Icons.wifi_off_rounded,
            title: "Can't reach your desktop",
            message: _error!,
            action: SizedBox(
              width: 200,
              child: BrandButton(
                label: 'Try again',
                icon: Icons.refresh_rounded,
                onPressed: _load,
              ),
            ),
          ),
        ],
      );
    }

    final status = _status!;
    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.fromLTRB(20, 4, 20, 32),
      children: [
        SectionCard(
          title: 'Desktop',
          trailing: const StatusPill(SyncStatus.synced, label: 'Online'),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('Device ID', style: SyncyText.muted),
              const SizedBox(height: 6),
              SelectableText(status.deviceId, style: SyncyText.monoStrong),
              const SizedBox(height: 18),
              Row(
                children: [
                  _StatTile(
                    icon: Icons.folder_rounded,
                    value: '${status.folders}',
                    label: status.folders == 1 ? 'Folder' : 'Folders',
                  ),
                  const SizedBox(width: 12),
                  _StatTile(
                    icon: Icons.devices_rounded,
                    value: '${status.devices}',
                    label: status.devices == 1 ? 'Device' : 'Devices',
                  ),
                ],
              ),
            ],
          ),
        ),
        if (_conflicts.isNotEmpty) ...[
          const SizedBox(height: 16),
          SectionCard(
            title: 'Needs attention',
            trailing: StatusPill(SyncStatus.error, label: '${_conflicts.length}'),
            child: Column(children: _conflictRows()),
          ),
        ],
        const SizedBox(height: 16),
        SectionCard(
          title: 'Devices',
          trailing: Text('${_devices.length}', style: SyncyText.muted),
          child: _devices.isEmpty
              ? const Text(
                  'No other devices yet. Pair one from Syncy on your desktop and it will appear here.',
                  style: SyncyText.muted,
                )
              : Column(children: _deviceRows()),
        ),
      ],
    );
  }

  List<Widget> _conflictRows() {
    final rows = <Widget>[];
    for (var i = 0; i < _conflicts.length; i++) {
      final conflict = _conflicts[i];
      if (i > 0) rows.add(const Divider(height: 24, color: SyncyColors.border));
      rows.add(
        Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Padding(
              padding: EdgeInsets.only(top: 4),
              child: StatusDot(SyncStatus.error, size: 8),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(conflict.path, style: SyncyText.monoStrong),
                  const SizedBox(height: 3),
                  Text('in ${conflict.folderId}', style: SyncyText.muted),
                ],
              ),
            ),
          ],
        ),
      );
    }
    return rows;
  }

  List<Widget> _deviceRows() {
    final rows = <Widget>[];
    for (var i = 0; i < _devices.length; i++) {
      final device = _devices[i];
      final status = device.trusted ? SyncStatus.synced : SyncStatus.pending;
      if (i > 0) rows.add(const Divider(height: 26, color: SyncyColors.border));
      rows.add(
        Row(
          children: [
            StatusDot(status),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(device.name, style: SyncyText.cardTitle),
                  const SizedBox(height: 3),
                  Text(
                    device.id,
                    style: SyncyText.mono,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ),
            ),
            const SizedBox(width: 12),
            StatusPill(status, label: device.trusted ? 'Trusted' : 'Pending'),
          ],
        ),
      );
    }
    return rows;
  }
}

class _StatTile extends StatelessWidget {
  const _StatTile({required this.icon, required this.value, required this.label});

  final IconData icon;
  final String value;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: SyncyColors.surfaceRaised,
          borderRadius: BorderRadius.circular(14),
          border: Border.all(color: SyncyColors.border),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(icon, color: SyncyColors.accent, size: 18),
            const SizedBox(height: 12),
            Text(value, style: SyncyText.stat),
            const SizedBox(height: 2),
            Text(label, style: SyncyText.muted),
          ],
        ),
      ),
    );
  }
}
