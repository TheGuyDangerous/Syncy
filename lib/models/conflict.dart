class Conflict {
  const Conflict({required this.folderId, required this.path});

  final String folderId;
  final String path;

  factory Conflict.fromJson(Map<String, dynamic> json) {
    return Conflict(
      folderId: json['folder_id'] as String? ?? '',
      path: json['path'] as String? ?? '',
    );
  }
}
