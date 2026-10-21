import React from "react";
import Link from "next/link";
import Logo from "@/components/brand/Logo";

export default function RegistroBrand() {
  return (
    <Link href="/" className="reg-brand" aria-label="fudd">
      <Logo mark="full" className="h-28 w-auto" />
    </Link>
  );
}
