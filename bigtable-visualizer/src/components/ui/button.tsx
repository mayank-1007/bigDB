import React from "react"

export function Button({

children,
className="",
...props

}:any){

return(

<button

className={`
rounded-2xl
px-5
py-3
bg-cyan-500
text-black
font-semibold
hover:scale-[1.02]
duration-300
${className}
`}

{...props}

>

{children}

</button>

)

}