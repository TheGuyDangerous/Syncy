class DeviceStatus {
  const DeviceStatus({
    required this.deviceId,
    required this.folders,
    required this.devices,
  });

  final String deviceId;
  final int folders;
  final int devices;

  factory DeviceStatus.fromJson(Map<String, dynamic> json) {
    return DeviceStatus(
      deviceId: json['device_id'] as String? ?? 'unknown',
      folders: (json['folders'] as num?)?.toInt() ?? 0,
      devices: (json['devices'] as num?)?.toInt() ?? 0,
    );
  }
}
