class Device {
  const Device({
    required this.id,
    required this.name,
    required this.trusted,
    this.lastSeen,
    this.addedAt,
  });

  final String id;
  final String name;
  final bool trusted;
  final String? lastSeen;
  final String? addedAt;

  factory Device.fromJson(Map<String, dynamic> json) {
    return Device(
      id: json['id'] as String? ?? '',
      name: json['name'] as String? ?? 'Unknown device',
      trusted: json['trusted'] as bool? ?? false,
      lastSeen: json['last_seen'] as String?,
      addedAt: json['added_at'] as String?,
    );
  }
}
