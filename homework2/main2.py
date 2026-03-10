from fastapi import FastAPI, HTTPException, UploadFile, File
from fastapi.responses import FileResponse
import os
from pydantic import BaseModel
from typing import Optional, Dict

app = FastAPI()


class Product(BaseModel):
    id: int
    name: str
    description: str
    icon: Optional[str] = None


class ProductCreate(BaseModel):
    name: str
    description: str


class ProductUpdate(BaseModel):
    id: Optional[int] = None
    name: Optional[str] = None
    description: Optional[str] = None


products: Dict[int, Product] = {}
next_id = 1
IMAGE_DIR = "images"
os.makedirs(IMAGE_DIR, exist_ok=True)


@app.post("/product", response_model=Product)
def create_product(product: ProductCreate):
    global next_id

    while next_id in products:
        next_id += 1

    new_product = Product(
        id=next_id,
        name=product.name,
        description=product.description,
    )

    products[next_id] = new_product
    next_id += 1

    return new_product


@app.get("/product/{product_id}", response_model=Product)
def get_product(product_id: int):
    if product_id not in products:
        raise HTTPException(status_code=404, detail="Product not found")

    return products[product_id]


@app.put("/product/{product_id}", response_model=Product)
def update_product(product_id: int, product_update: ProductUpdate):

    if product_id not in products:
        raise HTTPException(status_code=404, detail="Product not found")

    product = products[product_id]

    if product_update.id is not None:
        if product_update.id != product_id and product_update.id in products:
            raise HTTPException(status_code=409, detail="Product with this id already exists")
        product.id = product_update.id

    if product_update.name is not None:
        product.name = product_update.name

    if product_update.description is not None:
        product.description = product_update.description

    products.pop(product_id)
    products[product.id] = product

    return product


@app.delete("/product/{product_id}", response_model=Product)
def delete_product(product_id: int):

    if product_id not in products:
        raise HTTPException(status_code=404, detail="Product not found")

    return products.pop(product_id)


@app.get("/products", response_model=list[Product])
def get_all_products():
    return list(products.values())


@app.post("/product/{product_id}/image")
async def upload_image(product_id: int, icon: UploadFile = File(...)):

    if product_id not in products:
        raise HTTPException(status_code=404, detail="Product not found")

    filename = f"{product_id}_{icon.filename}"
    filepath = os.path.join(IMAGE_DIR, filename)

    with open(filepath, "wb") as buffer:
        buffer.write(await icon.read())

    products[product_id].icon = filename

    return {"filename": filename}


@app.get("/product/{product_id}/image")
def get_image(product_id: int):

    if product_id not in products:
        raise HTTPException(status_code=404, detail="Product not found")

    product = products[product_id]

    if not product.icon:
        raise HTTPException(status_code=404, detail="Image not found")

    filepath = os.path.join(IMAGE_DIR, product.icon)

    if not os.path.exists(filepath):
        raise HTTPException(status_code=404, detail="Image not found")

    return FileResponse(filepath)
