class Folder {
  const Folder({
    required this.id,
    required this.label,
    required this.path,
    required this.direction,
    required this.paused,
    this.addedAt,
  });

  final String id;
  final String label;
  final String path;
  final String direction;
  final bool paused;
  final String? addedAt;

  factory Folder.fromJson(Map<String, dynamic> json) {
    return Folder(
      id: json['id'] as String? ?? '',
      label: json['label'] as String? ?? 'Untitled folder',
      path: json['path'] as String? ?? '',
      direction: json['direction'] as String? ?? 'two-way',
      paused: json['paused'] as bool? ?? false,
      addedAt: json['added_at'] as String?,
    );
  }
}
