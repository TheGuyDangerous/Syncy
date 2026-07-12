import 'package:shared_preferences/shared_preferences.dart';

class Connection {
  const Connection({required this.baseUrl, required this.token});

  final String baseUrl;
  final String token;
}

class ConnectionStore {
  static const String _baseUrlKey = 'syncy.base_url';
  static const String _tokenKey = 'syncy.token';
  static const String _pairedKey = 'syncy.paired';
  static const String _sharedKey = 'syncy.shared_folders';

  Future<Connection?> load() async {
    final prefs = await SharedPreferences.getInstance();
    if (prefs.getBool(_pairedKey) != true) return null;
    final baseUrl = prefs.getString(_baseUrlKey);
    final token = prefs.getString(_tokenKey);
    if (baseUrl == null || token == null) return null;
    return Connection(baseUrl: baseUrl, token: token);
  }

  Future<void> save(Connection connection) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_baseUrlKey, connection.baseUrl);
    await prefs.setString(_tokenKey, connection.token);
    await prefs.setBool(_pairedKey, true);
  }

  Future<void> clear() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_baseUrlKey);
    await prefs.remove(_tokenKey);
    await prefs.remove(_pairedKey);
  }

  Future<Set<String>> loadSharedFolders() async {
    final prefs = await SharedPreferences.getInstance();
    return (prefs.getStringList(_sharedKey) ?? const <String>[]).toSet();
  }

  Future<void> saveSharedFolders(Set<String> keys) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setStringList(_sharedKey, keys.toList());
  }
}
