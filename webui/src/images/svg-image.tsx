import React from "react"
import { ReactSVG } from "react-svg"

export interface SVGImageProps {
  src: string
  alt: string
  svgClassName?: string
  wrapperClassName?: string
}

export const SVGImage: React.FC<SVGImageProps> = ({
  src,
  alt,
  svgClassName,
  wrapperClassName,
}) => (
  <ReactSVG
    role="img"
    aria-label={alt}
    src={src}
    className={wrapperClassName}
    beforeInjection={(svg) => {
      svg.classList.add(svgClassName || "")
    }}
    wrapper="div"
  />
)
