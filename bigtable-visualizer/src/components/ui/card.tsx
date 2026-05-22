import React from "react"

export function Card({
className="",
children
}:any){

return(

<div
className={`
rounded-3xl
border
border-cyan-900/40
bg-[#000000]
backdrop-blur-xl
shadow-[0_0_80px_rgba(0,180,255,.08)]
${className}
`}
>

{children}

</div>

)

}

export function CardHeader({
children
}:any){

return(

<div className="p-6">

{children}

</div>

)

}

export function CardTitle({
children
}:any){

return(

<h2
className="
text-cyan-100
font-semibold
text-xl
"
>

{children}

</h2>

)

}

export function CardContent({
children
}:any){

return(

<div className="p-6">

{children}

</div>

)

}