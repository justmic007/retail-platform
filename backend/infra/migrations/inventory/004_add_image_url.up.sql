-- 004_add_image_url.up.sql
-- Adds image_url column and sets Cloudinary URLs for all seeded products.

ALTER TABLE products ADD COLUMN IF NOT EXISTS image_url TEXT;

UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661355/MILK-CHO-50G_oneozw.jpg'         WHERE sku = 'MILK-CHO-50G';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661313/WINE-FRUIT-1L_ajmybx.jpg'        WHERE sku = 'WINE-FRUIT-1L';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661308/YOGURT-SKIM-1L_n9c7qh.jpg'       WHERE sku = 'YOGURT-SKIM-1L';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661306/MILK-HONEY-1L_l5lgnu.jpg'        WHERE sku = 'MILK-HONEY-1L';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661301/WIG-GOLD-1KG_etfhms.jpg'         WHERE sku = 'WIG-GOLD-1KG';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661301/WINE-ALC-2L_f0fmmf.jpg'          WHERE sku = 'WINE-ALC-2L';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661300/SAUCE-RAGU-1KG_yilyhj.jpg'       WHERE sku = 'SAUCE-RAGU-1KG';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661299/RICE-BES-5KG_qhhvi9.jpg'         WHERE sku = 'RICE-BES-5KG';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661295/OIL-VEGE-3L_l1q7pc.jpg'          WHERE sku = 'OIL-VEGE-3L';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661281/SARDINE-FISH-1kG_sxufha.jpg'     WHERE sku = 'SARDINE-FISH-1KG';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661279/POT-SMALL-5L_vrmi3e.jpg'         WHERE sku = 'POT-SMALL-5L';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661275/PEFUME-WHITE-1L_mgsxtl.jpg'      WHERE sku = 'PEFUME-WHITE-1L';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661275/OLIVE_OIL-2L_rgm3cp.jpg'         WHERE sku = 'OLIVE-OIL-2L';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661267/PEFUME-BLUE-1L_djirph.jpg'       WHERE sku = 'PEFUME-BLUE-1L';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661266/MILK-LOWFAT-1L_lbele2.jpg'       WHERE sku = 'MILK-LOWFAT-1L';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661247/MILK-TONE-1L_syzvfm.jpg'         WHERE sku = 'MILK-TONE-1L';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661223/CUTLERY-WOODEN-10PCS_sfvgpj.jpg' WHERE sku = 'CUTLERY-WOODEN-10PCS';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661136/LOTION-BODY-1KG_fhm27b.jpg'      WHERE sku = 'LOTION-BODY-1KG';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661135/MAIZE-BAG-25KG_wzb4n2.jpg'       WHERE sku = 'MAIZE-BAG-25KG';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661131/BUTTER-CHERRYCOLD-1KG_o44btt.jpg' WHERE sku = 'BUTTER-CHERRYCOLD-1KG';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661098/BOTTLE-WATER-1L_xbcfmj.jpg'      WHERE sku = 'BOTTLE-WATER-1L';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661123/EGG-CRATE-30_akktdu.jpg'         WHERE sku = 'EGG-CRATE-30';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661111/COFFEE-HOT-1L_ktrfnf.jpg'        WHERE sku = 'COFFEE-HOT-1L';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661106/BREAD-BRW-1KG_sdtsmn.jpg'        WHERE sku = 'BREAD-BRW-1KG';
UPDATE products SET image_url = 'https://res.cloudinary.com/dounyom8f/image/upload/q_auto/f_auto/v1778661096/EGG-CRATE-5_x1bemx.jpg'          WHERE sku = 'EGG-CRATE-5';
