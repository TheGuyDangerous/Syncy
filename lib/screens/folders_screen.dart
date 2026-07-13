import 'package:flutter/material.dart';

import '../models/folder.dart';
import '../services/api_client.dart';
import '../services/connection_store.dart';
import '../theme/app_theme.dart';
import '../widgets/page_header.dart';
import '../widgets/section_card.dart';
import '../widgets/status_indicators.dart';

class _ShareOption {
  const _ShareOption(this.id, this.label, this.subtitle, this.icon);

  final String id;
  final String label;
  final String subtitle;
  final IconData icon;
}

const List<_ShareOption> _shareOptions = [
  _ShareOption('photos', 'Photos', 'Camera roll and screenshots', Icons.photo_library_rounded),
  _ShareOption('downloads', 'Downloads', 'Files saved from other apps', Icons.download_rounded),
  _ShareOption('documents', 'Documents', 'PDFs, notes, and docs', Icons.description_rounded),
  _ShareOption('custom', 'Custom folder', 'Pick any folder to preview', Icons.create_new_folder_rounded),
];

class FoldersScreen extends StatefulWidget {
  const FoldersScreen({super.key, required this.api});

  final SyncyApi api;

  @override
  State<FoldersScreen> createState() => _FoldersScreenState();
}

class _FoldersScreenState extends State<FoldersScreen> {
  final ConnectionStore _store = ConnectionStore();
  bool _loading = true;
  String? _error;
  List<Folder> _folders = const [];
  Set<String> _selected = <String>{};

  @override
  void initState() {
    super.initState();
    _loadShared();
    _load();
  }

  Future<void> _loadShared() async {
    final selected = await _store.loadSharedFolders();
    if (!mounted) return;
    setState(() => _selected = selected);
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final folders = await widget.api.folders();
      if (!mounted) return;
      setState(() {
        _folders = folders;
        _loading = false;
      });
    } on ApiException catch (e) {
      if (!mounted) return;
      final hadData = _folders.isNotEmpty;
      setState(() {
        _error = e.message;
        _loading = false;
      });
      if (hadData) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.message)),
        );
      }
    }
  }

  Future<void> _toggleShare(String id, bool on) async {
    setState(() {
      final next = Set<String>.from(_selected);
      if (on) {
        next.add(id);
      } else {
        next.remove(id);
      }
      _selected = next;
    });
    await _store.saveSharedFolders(_selected);
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      bottom: false,
      child: Column(
        children: [
          PageHeader(
            title: 'Folders',
            subtitle: 'What your desktop is syncing',
            trailing: RefreshButton(loading: _loading, onTap: _load),
          ),
          Expanded(
            child: RefreshIndicator(
              onRefresh: _load,
              color: SyncyColors.accent,
              backgroundColor: SyncyColors.surfaceRaised,
              child: ListView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.fromLTRB(20, 4, 20, 32),
                children: [
                  SectionCard(
                    title: 'Desktop folders',
                    trailing: _folders.isEmpty
                        ? null
                        : Text('${_folders.length}', style: SyncyText.muted),
                    child: _foldersBody(),
                  ),
                  const SizedBox(height: 16),
                  SectionCard(
                    title: 'Share from this phone',
                    child: _shareBody(),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _foldersBody() {
    if (_loading && _folders.isEmpty) {
      return const Padding(
        padding: EdgeInsets.symmetric(vertical: 12),
        child: Center(child: CircularProgressIndicator()),
      );
    }
    if (_error != null && _folders.isEmpty) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(_error!, style: SyncyText.muted),
          const SizedBox(height: 12),
          TextButton(onPressed: _load, child: const Text('Try again')),
        ],
      );
    }
    if (_folders.isEmpty) {
      return const Text(
        'No folders yet. Add one in Syncy on your desktop and it will show up here.',
        style: SyncyText.muted,
      );
    }
    return Column(children: _folderRows());
  }

  List<Widget> _folderRows() {
    final rows = <Widget>[];
    for (var i = 0; i < _folders.length; i++) {
      final folder = _folders[i];
      final status = folder.paused ? SyncStatus.paused : SyncStatus.synced;
      if (i > 0) rows.add(const Divider(height: 24, color: SyncyColors.border));
      rows.add(
        Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.only(top: 4),
              child: StatusDot(status),
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(child: Text(folder.label, style: SyncyText.cardTitle)),
                      const SizedBox(width: 10),
                      StatusPill(status, label: folder.paused ? 'Paused' : 'Synced'),
                    ],
                  ),
                  const SizedBox(height: 6),
                  Text(
                    folder.path,
                    style: SyncyText.mono,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  const SizedBox(height: 10),
                  _DirectionChip(folder.direction),
                ],
              ),
            ),
          ],
        ),
      );
    }
    return rows;
  }

  Widget _shareBody() {
    final rows = <Widget>[];
    for (var i = 0; i < _shareOptions.length; i++) {
      final option = _shareOptions[i];
      if (i > 0) rows.add(const Divider(height: 24, color: SyncyColors.border));
      rows.add(_shareRow(option));
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: SyncyColors.accentSoft,
            borderRadius: BorderRadius.circular(12),
          ),
          child: const Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(Icons.info_outline_rounded, color: SyncyColors.accent, size: 18),
              SizedBox(width: 10),
              Expanded(
                child: Text(
                  'Preview — on-device sync is coming. Choose what you would share; picks stay on this phone and nothing uploads yet.',
                  style: SyncyText.muted,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),
        ...rows,
      ],
    );
  }

  Widget _shareRow(_ShareOption option) {
    final selected = _selected.contains(option.id);
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: () => _toggleShare(option.id, !selected),
        borderRadius: BorderRadius.circular(12),
        child: Row(
          children: [
            Container(
              width: 40,
              height: 40,
              alignment: Alignment.center,
              decoration: BoxDecoration(
                color: SyncyColors.surfaceRaised,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: SyncyColors.border),
              ),
              child: Icon(option.icon, color: SyncyColors.accent, size: 20),
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(option.label, style: SyncyText.cardTitle),
                  const SizedBox(height: 2),
                  Text(option.subtitle, style: SyncyText.muted),
                ],
              ),
            ),
            Switch(
              value: selected,
              onChanged: (value) => _toggleShare(option.id, value),
            ),
          ],
        ),
      ),
    );
  }
}

class _DirectionChip extends StatelessWidget {
  const _DirectionChip(this.direction);

  final String direction;

  @override
  Widget build(BuildContext context) {
    final (icon, label) = _describe(direction);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 5),
      decoration: BoxDecoration(
        color: SyncyColors.surfaceRaised,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: SyncyColors.border),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, color: SyncyColors.muted, size: 14),
          const SizedBox(width: 6),
          Text(
            label,
            style: const TextStyle(
              color: SyncyColors.muted,
              fontSize: 12,
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }

  (IconData, String) _describe(String direction) {
    switch (direction.toLowerCase()) {
      case 'send':
      case 'push':
      case 'to':
      case 'send-only':
      case 'one-way':
        return (Icons.arrow_upward_rounded, 'Send');
      case 'receive':
      case 'pull':
      case 'from':
      case 'receive-only':
        return (Icons.arrow_downward_rounded, 'Receive');
      default:
        return (Icons.sync_alt_rounded, 'Two-way');
    }
  }
}
