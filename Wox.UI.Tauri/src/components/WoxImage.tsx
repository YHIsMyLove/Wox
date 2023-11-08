import { WoxImageTypeEnum } from "../enums/WoxImageTypeEnum.ts"
import { WOXMESSAGE } from "../entity/WoxMessage.typings"
import styled from "styled-components"

export default (props: { img: WOXMESSAGE.WoxImage, height: number, width: number }) => {
  return <Style width={props.width} height={props.height}>
    {props.img.ImageType === WoxImageTypeEnum.WoxImageTypeSvg.code &&
      <div className={"wox-image"} dangerouslySetInnerHTML={{ __html: props.img.ImageData }}></div>}
    {props.img.ImageType === WoxImageTypeEnum.WoxImageTypeUrl.code &&
      <img src={props.img.ImageData} className={"wox-image"} alt={"wox-image"} />}
    {props.img.ImageType === WoxImageTypeEnum.WoxImageTypeBase64.code &&
      <img src={props.img.ImageData} className={"wox-image"} alt={"wox-image"} />}
  </Style>
}

const Style = styled.div<{ width: number; height: number }>`
  .wox-image {
    line-height: ${props => props.height}px;
    height: ${props => props.height}px;
    width: ${props => props.width}px;
    text-align: center;
    vertical-align: middle;
    
    svg {
      width: ${props => props.height}px !important;
      height: ${props => props.height}px !important;
    }
  }
`