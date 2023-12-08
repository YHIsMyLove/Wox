import 'package:wox/enums/wox_image_type_enum.dart';

class WoxImage {
  late WoxImageType imageType;
  late String imageData;

  WoxImage({required this.imageType, required this.imageData});

  WoxImage.fromJson(Map<String, dynamic> json) {
    imageType = json['ImageType'];
    imageData = json['ImageData'];
  }

  Map<String, dynamic> toJson() {
    final Map<String, dynamic> data = <String, dynamic>{};
    data['ImageType'] = imageType;
    data['ImageData'] = imageData;
    return data;
  }

  static WoxImage? parse(String imageData) {
    //split image data with : to get image type, only get first part
    final List<String> imageDataList = imageData.split(':');
    if (imageDataList.length < 2) return null;

    final imageType = imageDataList[0];
    // the rest of the string is the image data
    final imageDataString = imageDataList.sublist(1).join(':');

    if (imageType == WoxImageTypeEnum.WOX_IMAGE_TYPE_BASE64.code) {
      return WoxImage(imageType: WoxImageTypeEnum.WOX_IMAGE_TYPE_BASE64.code, imageData: imageDataString);
    } else if (imageType == WoxImageTypeEnum.WOX_IMAGE_TYPE_SVG.code) {
      return WoxImage(imageType: WoxImageTypeEnum.WOX_IMAGE_TYPE_SVG.code, imageData: imageDataString);
    } else if (imageType == WoxImageTypeEnum.WOX_IMAGE_TYPE_URL.code) {
      return WoxImage(imageType: WoxImageTypeEnum.WOX_IMAGE_TYPE_URL.code, imageData: imageDataString);
    } else if (imageType == WoxImageTypeEnum.WOX_IMAGE_TYPE_ABSOLUTE_PATH.code) {
      return WoxImage(imageType: WoxImageTypeEnum.WOX_IMAGE_TYPE_ABSOLUTE_PATH.code, imageData: imageDataString);
    } else if (imageType == WoxImageTypeEnum.WOX_IMAGE_TYPE_RELATIVE_PATH.code) {
      return WoxImage(imageType: WoxImageTypeEnum.WOX_IMAGE_TYPE_RELATIVE_PATH.code, imageData: imageDataString);
    } else {
      return null;
    }
  }
}
