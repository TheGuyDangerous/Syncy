import 'package:flutter/material.dart';

import 'screens/folders_screen.dart';
import 'screens/history_screen.dart';
import 'screens/home_screen.dart';
import 'screens/pair_screen.dart';
import 'screens/settings_screen.dart';
import 'services/api_client.dart';
import 'services/connection_store.dart';
import 'theme/app_theme.dart';
import 'widgets/brand_mark.dart';

class SyncyApp extends StatelessWidget {
  const SyncyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Syncy',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.dark,
      home: const RootGate(),
    );
  }
}

class RootGate extends StatefulWidget {
  const RootGate({super.key});

  @override
  State<RootGate> createState() => _RootGateState();
}

class _RootGateState extends State<RootGate> {
  final ConnectionStore _store = ConnectionStore();
  bool _loading = true;
  SyncyApi? _api;

  @override
  void initState() {
    super.initState();
    _restore();
  }

  Future<void> _restore() async {
    final connection = await _store.load();
    if (!mounted) return;
    setState(() {
      _api = connection == null
          ? null
          : SyncyApi(baseUrl: connection.baseUrl, token: connection.token);
      _loading = false;
    });
  }

  void _onPaired(SyncyApi api) {
    setState(() => _api = api);
  }

  Future<void> _onUnpair() async {
    await _store.clear();
    if (!mounted) return;
    setState(() => _api = null);
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) return const _Splash();
    final api = _api;
    if (api == null) {
      return PairScreen(store: _store, onPaired: _onPaired);
    }
    return HomeShell(api: api, onUnpair: _onUnpair);
  }
}

class _Splash extends StatelessWidget {
  const _Splash();

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      backgroundColor: SyncyColors.background,
      body: Center(child: BrandMark(size: 64)),
    );
  }
}

class HomeShell extends StatefulWidget {
  const HomeShell({super.key, required this.api, required this.onUnpair});

  final SyncyApi api;
  final Future<void> Function() onUnpair;

  @override
  State<HomeShell> createState() => _HomeShellState();
}

class _HomeShellState extends State<HomeShell> {
  int _index = 0;

  @override
  Widget build(BuildContext context) {
    final screens = [
      HomeScreen(api: widget.api),
      FoldersScreen(api: widget.api),
      const HistoryScreen(),
      SettingsScreen(api: widget.api, onUnpair: widget.onUnpair),
    ];

    return Scaffold(
      body: IndexedStack(index: _index, children: screens),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _index,
        onDestinationSelected: (value) => setState(() => _index = value),
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.dashboard_outlined),
            selectedIcon: Icon(Icons.dashboard_rounded),
            label: 'Home',
          ),
          NavigationDestination(
            icon: Icon(Icons.folder_outlined),
            selectedIcon: Icon(Icons.folder_rounded),
            label: 'Folders',
          ),
          NavigationDestination(
            icon: Icon(Icons.timeline_outlined),
            selectedIcon: Icon(Icons.timeline_rounded),
            label: 'History',
          ),
          NavigationDestination(
            icon: Icon(Icons.settings_outlined),
            selectedIcon: Icon(Icons.settings_rounded),
            label: 'Settings',
          ),
        ],
      ),
    );
  }
}
