import 'package:flutter/material.dart';

import '../theme/app_theme.dart';

class BrandMark extends StatelessWidget {
  const BrandMark({super.key, this.size = 40});

  final double size;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: size,
      height: size,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        gradient: SyncyColors.brandGradient,
        borderRadius: BorderRadius.circular(size * 0.28),
        boxShadow: [
          BoxShadow(
            color: SyncyColors.brandGlow,
            blurRadius: size * 0.5,
            offset: Offset(0, size * 0.16),
          ),
        ],
      ),
      child: Text(
        'S',
        style: TextStyle(
          color: Colors.white,
          fontSize: size * 0.56,
          fontWeight: FontWeight.w800,
          height: 1,
          letterSpacing: -0.5,
        ),
      ),
    );
  }
}
