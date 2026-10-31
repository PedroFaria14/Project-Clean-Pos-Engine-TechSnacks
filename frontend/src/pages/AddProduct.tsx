import { ProductForm, type ProductFormProps } from "../components/ProductForm";

export function AddProduct(props: Omit<ProductFormProps, "product">) {
  return <ProductForm {...props} />;
}
