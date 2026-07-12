import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:http/http.dart' as http;

import '../models/conflict.dart';
import '../models/device.dart';
import '../models/device_status.dart';
import '../models/folder.dart';

class ApiException implements Exception {
  const ApiException(this.message);

  final String message;

  @override
  String toString() => message;
}

class SyncyApi {
  SyncyApi({required this.baseUrl, required this.token, http.Client? client})
      : _client = client ?? http.Client();

  final String baseUrl;
  final String token;
  final http.Client _client;

  static const Duration _timeout = Duration(seconds: 8);

  Future<DeviceStatus> status() async {
    final data = await _get('/status');
    if (data is! Map<String, dynamic>) {
      throw const ApiException(
        'That address answered, but not like Syncy would. Double-check the host and port.',
      );
    }
    return DeviceStatus.fromJson(data);
  }

  Future<List<Folder>> folders() async {
    return _listOf('/folders', Folder.fromJson);
  }

  Future<List<Device>> devices() async {
    return _listOf('/devices', Device.fromJson);
  }

  Future<List<Conflict>> conflicts() async {
    return _listOf('/conflicts', Conflict.fromJson);
  }

  Future<List<T>> _listOf<T>(
    String path,
    T Function(Map<String, dynamic>) parse,
  ) async {
    final data = await _get(path);
    if (data is! List) return const [];
    return data.whereType<Map<String, dynamic>>().map(parse).toList();
  }

  Future<dynamic> _get(String path) async {
    final http.Response response;
    try {
      response = await _client
          .get(_resolve(path), headers: {'Authorization': 'Bearer $token'})
          .timeout(_timeout);
    } on TimeoutException {
      throw const ApiException(
        'Your desktop took too long to answer. Make sure it is awake and on the same network.',
      );
    } on SocketException {
      throw const ApiException(
        "Couldn't reach your desktop. Check the address and that Syncy is running.",
      );
    } catch (_) {
      throw const ApiException(
        "Couldn't reach your desktop. Check the address and that Syncy is running.",
      );
    }

    if (response.statusCode == 401 || response.statusCode == 403) {
      throw const ApiException(
        'That access token was rejected. Copy a fresh one from Syncy on your desktop.',
      );
    }
    if (response.statusCode >= 400) {
      throw ApiException(
        'Your desktop returned an error (${response.statusCode}). Try again in a moment.',
      );
    }

    try {
      return jsonDecode(response.body);
    } catch (_) {
      throw const ApiException(
        'Got an unexpected response. Make sure the address points to Syncy, not another app.',
      );
    }
  }

  Uri _resolve(String path) {
    final trimmed = baseUrl.trim().replaceAll(RegExp(r'/+$'), '');
    final withScheme = trimmed.startsWith('http://') || trimmed.startsWith('https://')
        ? trimmed
        : 'http://$trimmed';
    return Uri.parse('$withScheme$path');
  }

  void close() => _client.close();
}
