import 'package:flutter/material.dart';

import '../services/api_client.dart';
import '../services/connection_store.dart';
import '../theme/app_theme.dart';
import '../widgets/brand_button.dart';
import '../widgets/brand_mark.dart';

class PairScreen extends StatefulWidget {
  const PairScreen({super.key, required this.store, required this.onPaired});

  final ConnectionStore store;
  final void Function(SyncyApi api) onPaired;

  @override
  State<PairScreen> createState() => _PairScreenState();
}

class _PairScreenState extends State<PairScreen> {
  final TextEditingController _addressController = TextEditingController();
  final TextEditingController _tokenController = TextEditingController();
  bool _connecting = false;
  bool _obscureToken = true;
  String? _error;

  @override
  void dispose() {
    _addressController.dispose();
    _tokenController.dispose();
    super.dispose();
  }

  Future<void> _connect() async {
    if (_connecting) return;
    final address = _addressController.text.trim();
    final token = _tokenController.text.trim();
    if (address.isEmpty || token.isEmpty) {
      setState(() => _error = 'Enter your desktop address and access token to connect.');
      return;
    }

    setState(() {
      _connecting = true;
      _error = null;
    });

    final api = SyncyApi(baseUrl: address, token: token);
    try {
      await api.status();
      await widget.store.save(Connection(baseUrl: address, token: token));
      if (!mounted) return;
      widget.onPaired(api);
    } on ApiException catch (e) {
      api.close();
      if (!mounted) return;
      setState(() {
        _error = e.message;
        _connecting = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 32),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 440),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Center(child: BrandMark(size: 64)),
                  const SizedBox(height: 24),
                  const Text(
                    'Pair with your desktop',
                    textAlign: TextAlign.center,
                    style: SyncyText.screenTitle,
                  ),
                  const SizedBox(height: 10),
                  const Text(
                    'Syncy on your phone is a companion. Point it at the Syncy app on your computer to watch and manage sync from your phone.',
                    textAlign: TextAlign.center,
                    style: SyncyText.muted,
                  ),
                  const SizedBox(height: 32),
                  const _FieldLabel('Desktop address'),
                  const SizedBox(height: 8),
                  TextField(
                    controller: _addressController,
                    keyboardType: TextInputType.url,
                    autocorrect: false,
                    textInputAction: TextInputAction.next,
                    style: SyncyText.monoStrong,
                    decoration: const InputDecoration(hintText: '192.168.1.20:22062'),
                  ),
                  const SizedBox(height: 20),
                  const _FieldLabel('Access token'),
                  const SizedBox(height: 8),
                  TextField(
                    controller: _tokenController,
                    obscureText: _obscureToken,
                    autocorrect: false,
                    enableSuggestions: false,
                    textInputAction: TextInputAction.done,
                    onSubmitted: (_) => _connect(),
                    style: SyncyText.monoStrong,
                    decoration: InputDecoration(
                      hintText: 'Paste the token from Syncy',
                      suffixIcon: IconButton(
                        icon: Icon(
                          _obscureToken ? Icons.visibility_rounded : Icons.visibility_off_rounded,
                          color: SyncyColors.muted,
                        ),
                        onPressed: () => setState(() => _obscureToken = !_obscureToken),
                      ),
                    ),
                  ),
                  if (_error != null) ...[
                    const SizedBox(height: 18),
                    _ErrorBanner(_error!),
                  ],
                  const SizedBox(height: 24),
                  BrandButton(
                    label: _connecting ? 'Connecting…' : 'Connect',
                    icon: Icons.link_rounded,
                    loading: _connecting,
                    onPressed: _connect,
                  ),
                  const SizedBox(height: 18),
                  const Text(
                    'Find the address and token in Syncy on your desktop, under Settings → Pair a phone.',
                    textAlign: TextAlign.center,
                    style: SyncyText.muted,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _FieldLabel extends StatelessWidget {
  const _FieldLabel(this.text);

  final String text;

  @override
  Widget build(BuildContext context) {
    return Text(text.toUpperCase(), style: SyncyText.eyebrow);
  }
}

class _ErrorBanner extends StatelessWidget {
  const _ErrorBanner(this.message);

  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: SyncyColors.dangerSoft,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: SyncyColors.danger),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(Icons.error_outline_rounded, color: SyncyColors.danger, size: 20),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              message,
              style: const TextStyle(color: SyncyColors.text, fontSize: 13, height: 1.4),
            ),
          ),
        ],
      ),
    );
  }
}
