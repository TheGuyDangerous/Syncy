import 'package:flutter/material.dart';

import '../services/api_client.dart';
import '../theme/app_theme.dart';
import '../widgets/brand_mark.dart';
import '../widgets/page_header.dart';
import '../widgets/section_card.dart';

const String _appVersion = '1.0.0';
const String _buildNumber = '1';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key, required this.api, required this.onUnpair});

  final SyncyApi api;
  final Future<void> Function() onUnpair;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      bottom: false,
      child: Column(
        children: [
          const PageHeader(title: 'Settings'),
          Expanded(
            child: ListView(
              padding: const EdgeInsets.fromLTRB(20, 4, 20, 32),
              children: [
                SectionCard(
                  title: 'Connection',
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      _Detail(label: 'Desktop address', value: _displayAddress(api.baseUrl)),
                      const Divider(height: 24, color: SyncyColors.border),
                      _Detail(label: 'Access token', value: _maskToken(api.token)),
                    ],
                  ),
                ),
                const SizedBox(height: 16),
                const SectionCard(
                  title: 'About',
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          BrandMark(size: 40),
                          SizedBox(width: 14),
                          Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text('Syncy Mobile', style: SyncyText.cardTitle),
                              SizedBox(height: 2),
                              Text(
                                'Version $_appVersion ($_buildNumber) · Companion preview',
                                style: SyncyText.muted,
                              ),
                            ],
                          ),
                        ],
                      ),
                      SizedBox(height: 16),
                      Text(
                        'A window into your Syncy desktop. On-device sync is on the way.',
                        style: SyncyText.muted,
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 16),
                _DisconnectButton(onConfirm: () => _confirmDisconnect(context)),
              ],
            ),
          ),
        ],
      ),
    );
  }

  String _displayAddress(String baseUrl) {
    final trimmed = baseUrl.trim().replaceAll(RegExp(r'/+$'), '');
    if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
      return trimmed;
    }
    return 'http://$trimmed';
  }

  String _maskToken(String token) {
    if (token.length <= 8) return '•' * (token.isEmpty ? 6 : token.length);
    return '${token.substring(0, 4)} •••• ${token.substring(token.length - 4)}';
  }

  Future<void> _confirmDisconnect(BuildContext context) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        backgroundColor: SyncyColors.surfaceRaised,
        title: const Text('Disconnect from desktop?'),
        content: const Text(
          "You'll need the address and access token to pair again.",
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, true),
            child: const Text(
              'Disconnect',
              style: TextStyle(color: SyncyColors.danger, fontWeight: FontWeight.w600),
            ),
          ),
        ],
      ),
    );
    if (confirmed == true) {
      api.close();
      await onUnpair();
    }
  }
}

class _Detail extends StatelessWidget {
  const _Detail({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label, style: SyncyText.muted),
        const SizedBox(height: 6),
        SelectableText(value, style: SyncyText.monoStrong),
      ],
    );
  }
}

class _DisconnectButton extends StatelessWidget {
  const _DisconnectButton({required this.onConfirm});

  final VoidCallback onConfirm;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onConfirm,
        borderRadius: BorderRadius.circular(14),
        child: Container(
          height: 54,
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: SyncyColors.dangerSoft,
            borderRadius: BorderRadius.circular(14),
            border: Border.all(color: SyncyColors.danger),
          ),
          child: const Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(Icons.link_off_rounded, color: SyncyColors.danger, size: 20),
              SizedBox(width: 10),
              Text(
                'Disconnect',
                style: TextStyle(
                  color: SyncyColors.danger,
                  fontSize: 15,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
