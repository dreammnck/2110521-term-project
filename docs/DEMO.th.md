# สคริปต์วิดีโอเดโม (เป้าหมาย ≈ 4:30 นาที, ห้ามเกิน 5:00)

English version: [DEMO.md](DEMO.md)

**ต้องแสดงให้ครบ:** Use case ทั้ง 3 ข้อ + Authentication · ฐานข้อมูล 2 ประเภท (**PostgreSQL = RDBMS**, **MongoDB = NoSQL**) · Service ที่ใช้ Message Broker (**RabbitMQ**)

## ก่อนกดอัดวิดีโอ (ประมาณ 5 นาที)

```bash
make clean && make up          # ล้างข้อมูลเก่า เริ่มใหม่ (~1 นาที หลัง build image แล้ว)
docker compose ps              # ทุก container ต้องเป็น Up / healthy
```
หลัง `up` ให้รอประมาณ 10 วินาทีก่อนเริ่มทำอะไร (ให้ Notification adapter เชื่อมต่อ RabbitMQ เสร็จ)

ถ้าอยากรัน `make smoke` เพื่อเช็กก่อน ให้รีเซ็ตข้อมูลอีกครั้งหลังรัน เพราะ smoke test ทิ้งข้อมูลทดสอบไว้:
`docker compose down -v && docker compose up -d`

จัดหน้าจอ (สลับหน้าต่างเอา ไม่ต้องเลื่อนหน้าจอ):

| หน้าต่าง | URL / คำสั่ง | หมายเหตุ |
|---|---|---|
| **A. Staff app** (ขนาดเดสก์ท็อป) | http://localhost:3000 (เครื่องนี้ใช้ **3100** ตามไฟล์ `.env`) | ยังไม่ล็อกอิน |
| **B. Resident app** (ขนาดมือถือ 390×844 ใช้ DevTools device mode หรือย่อหน้าต่าง) | http://localhost:3001 (เครื่องนี้ใช้ **3101**) | ยังไม่ล็อกอิน |
| **C. RabbitMQ UI** | http://localhost:15672 → ล็อกอิน `guest`/`guest` → **Queues → notification-adapter** | เปิดค้างไว้ |
| **D. Terminal** (ขยายตัวอักษรให้ใหญ่) | โฟลเดอร์โปรเจกต์ | แบ่ง 2 pane: pane แรกใช้ `psql`/`mongosh`, pane สองใช้ดู log |

เตรียมรูปภาพขนาดเล็กไว้ 1 รูป (`~/Desktop/parcel.jpg`) สำหรับแนบรูปพัสดุ
เปิด log ค้างไว้ใน pane ที่สองก่อนเริ่มอัด:
```bash
docker compose logs -f notification-adapter | grep --line-buffered "provider: send"
```

บัญชีสำหรับเดโม (รหัสผ่านทุกบัญชี `password123`):
`staff@condo.local`, `resident1@condo.local` (ห้อง A101 อาคาร A), `resident7@condo.local` (อาคาร B)

---

## สคริปต์

### 0:00 – 0:25 · แนะนำระบบ (เปิดแผนภาพสถาปัตยกรรมใน README)
> "ระบบนี้คือระบบบริหารจัดการคอนโดมิเนียม พัฒนาด้วย Go แบบ Microservices และ Next.js สำหรับหน้าบ้าน ผู้ใช้มี 2 กลุ่มคือลูกบ้านและเจ้าหน้าที่นิติบุคคล ใช้จัดการพัสดุ ค่าน้ำค่าไฟ และประกาศจากส่วนกลาง แต่ละ Service มีฐานข้อมูลของตัวเอง ข้อมูลเชิงสัมพันธ์เก็บใน PostgreSQL ส่วนการแจ้งเตือนเก็บใน MongoDB และใช้ RabbitMQ ส่งอีเวนต์ระหว่าง Service"

ชี้ที่แผนภาพ: Gateway → Services → **PostgreSQL** (identity, parcel, announcement, billing) / **MongoDB** (notification) / **RabbitMQ**

### 0:25 – 0:55 · Authentication (การยืนยันตัวตน)
1. **หน้าต่าง A (staff):** ลองล็อกอินด้วย `resident1@condo.local` → ขึ้นข้อความสีแดง *"This app is for juristic staff only"*
   > "ระบบตรวจสอบบทบาทผู้ใช้ ลูกบ้านไม่สามารถเข้าแอปของเจ้าหน้าที่ได้"
2. ล็อกอินด้วย `staff@condo.local` → เข้าสู่ Dashboard
3. **หน้าต่าง B (resident):** ล็อกอินด้วย `resident1@condo.local` → หน้า Home
   > "การล็อกอินผ่าน Identity Service ซึ่งออก JWT ให้ แล้ว API Gateway จะตรวจสอบ token และส่งข้อมูลผู้ใช้ต่อไปยัง Service ปลายทาง"

### 0:55 – 1:50 · Use case 1: การจัดการและรับพัสดุ (+ RabbitMQ)
1. **หน้าต่าง C:** แสดง queue `notification-adapter` มี 1 consumer และยังไม่มีข้อความค้าง
2. **หน้าต่าง A → เมนู Parcels:** เลือก Building A ห้อง **A101** ผู้ให้บริการ *Kerry* เลขพัสดุ `TH123` แนบรูป → กด **Register parcel**
3. **หน้าต่าง C:** กราฟ message rate ขยับ (published / acked)
   > "Parcel Service ส่งอีเวนต์ `ParcelReceived` เข้า RabbitMQ แล้ว Notification adapter รับอีเวนต์ไปประมวลผล"
4. **Terminal (pane log):** จะเห็นบรรทัด `notification provider: send … New parcel arrived`
5. **หน้าต่าง B:** จุดแดงที่กระดิ่งปรากฏ (ดึงหน้าจอลงเพื่อรีเฟรช หรือรอไม่เกิน 15 วินาที) → เปิด **Parcels** → เห็นรูปและสถานะ *Waiting* → กด **Confirm pickup** → ใน bottom sheet กด *Yes, I have it*
6. **หน้าต่าง A:** พัสดุเปลี่ยนเป็น *Picked up* และจำนวนบน Dashboard ลดลง
   > "เมื่อลูกบ้านยืนยันรับพัสดุ ระบบส่งอีเวนต์ `ParcelAccepted` เพื่อแจ้งเจ้าหน้าที่"

### 1:50 – 2:55 · Use case 2: การตรวจสอบและชำระค่าน้ำค่าไฟ
1. **หน้าต่าง A → เมนู Utility bills → Single room:** เลือกห้อง A101 รอบบิลเดือนปัจจุบัน หน่วยน้ำ `10` หน่วยไฟ `100` ครบกำหนดอีก 14 วัน → กด **Issue bill** (ยอดรวม ฿880.00)
2. **หน้าต่าง B → เมนู Bills:** เห็นบิลใหม่ ฿880 สถานะ *Unpaid* (และมีแจ้งเตือนบิลใหม่) → เปิดบิล → กด **Pay ฿880.00** → *Continue to payment*
3. หน้า Payment Gateway จำลองเปิดขึ้น → กด **Pay now**
   > "Payment Gateway เป็นระบบภายนอก เมื่อชำระเสร็จ Gateway จะเรียก Payment Adapter ของเราผ่าน Webhook ที่เซ็นด้วย HMAC และ Adapter จะตรวจสอบลายเซ็นก่อนให้ Billing Service อัปเดตสถานะบิลเป็นชำระแล้ว"
4. ระบบพากลับมาที่แอป: *Payment successful* → กด **View receipt** (เห็นเลขใบเสร็จและเลขอ้างอิงการชำระเงิน)
5. **หน้าต่าง A → Utility bills:** บิลเดียวกันเปลี่ยนเป็น *Paid*

### 2:55 – 3:30 · Use case 3: การเผยแพร่และรับประกาศจากนิติบุคคล
1. **หน้าต่าง A → เมนู Announcements → New:** หัวข้อ *Lift maintenance* กลุ่มเป้าหมาย **A specific building → Building A** → กด **Publish**
2. **หน้าต่าง B → เมนู News:** ประกาศปรากฏพร้อมป้าย *New* → เปิดอ่าน (ระบบบันทึกว่าอ่านแล้ว)
3. แสดงความต่าง (Terminal ประมาณ 10 วินาที) ลูกบ้านอาคาร B มองไม่เห็นประกาศนี้:
   ```bash
   T=$(curl -s -X POST localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"resident7@condo.local","password":"password123"}' | jq -r .token)
   curl -s localhost:8080/api/v1/announcements -H "Authorization: Bearer $T"      # → []
   ```
   > "การกำหนดกลุ่มเป้าหมายทำที่ Announcement Service ซึ่ง Gateway เรียกผ่าน gRPC"

### 3:30 – 4:30 · หลักฐานฐานข้อมูล 2 ประเภท + Message Broker (Terminal)
**RDBMS – PostgreSQL** (2 คำสั่งสั้น ๆ):
```bash
docker compose exec postgres psql -U condo -d parcel_db  -c "select id, carrier, status, received_at, picked_up_at from parcels order by received_at desc limit 3;"
docker compose exec postgres psql -U condo -d billing_db -c "select b.cycle, b.total, b.status, p.payment_reference, p.payment_status from utility_bills b join payments p on p.bill_id=b.id order by p.created_at desc limit 3;"
```
> "ข้อมูลเชิงสัมพันธ์ใช้ Foreign Key เช่น การชำระเงินต้องอ้างอิงบิลที่มีอยู่จริง"

**NoSQL – MongoDB** (การแจ้งเตือน):
```bash
docker compose exec mongo mongosh notification_db --quiet --eval 'db.notifications.find({}, {title:1, type:1, read_at:1, created_at:1}).sort({created_at:-1}).limit(5).forEach(printjson); printjson(db.notifications.getIndexes().map(i => i.name))'
```
> "การแจ้งเตือนเก็บเป็น Document ใน MongoDB และมี Unique Index บน event id กับ user id ทำให้ถ้าอีเวนต์ถูกส่งซ้ำก็ไม่เกิดการแจ้งเตือนซ้ำ"

**RabbitMQ:** สลับไปหน้าต่าง C → **Exchanges → `condo.events`** (ประเภท topic) และ **Queues → `notification-adapter`** → ชี้ตัวเลข *Total / Ack*
> "Parcel, Announcement และ Billing Service ส่งอีเวนต์เข้า Exchange นี้ และ Notification adapter เป็นผู้รับ"

### 4:30 – 4:50 · สรุป
> "สรุปคือระบบรองรับ Use case ครบทั้ง 3 ข้อ มีการยืนยันตัวตนด้วย JWT และแยกสิทธิ์ตามบทบาท ใช้ฐานข้อมูล 2 ประเภทคือ PostgreSQL และ MongoDB และใช้ RabbitMQ สำหรับการแจ้งเตือนแบบ Event-driven ซึ่งสอดคล้องกับ ADR ที่เราเลือกไว้ คือ Microservices, REST ร่วมกับ gRPC, Event-driven Notification, Relational Database และการเชื่อมต่อ Payment Gateway ภายนอก"

---

## เคล็ดลับเรื่องเวลา
- ซ้อมสัก 1 รอบก่อนอัดจริง จุดที่ช้าที่สุดคือการอัปโหลดรูปและการเด้งไปหน้าชำระเงิน ให้ใช้รูปไฟล์เล็ก ๆ
- ถ้าจุดแจ้งเตือนขึ้นช้า ให้ **ดึงหน้าจอลงเพื่อรีเฟรช** ในหน้าต่าง resident (หรือสลับแท็บไปมา ข้อมูลจะรีเฟรชเองเมื่อกลับมา)
- ช่วงที่รอให้ตัดออกตอนตัดต่อ แทนการพูดเร็วขึ้น

## ถ้ามีปัญหาในวันอัดจริง
| อาการ | วิธีแก้ |
|---|---|
| เปิดหน้าเว็บไม่ได้ | `docker compose ps` แล้ว `docker compose up -d` |
| พอร์ต 3000/3001 ถูกใช้อยู่แล้ว | ใส่ `STAFF_PORT=3100` / `RESIDENT_PORT=3101` ในไฟล์ `.env` แล้ว `docker compose up -d` |
| ไม่มีการแจ้งเตือน | `docker compose logs notification-adapter` (ต้องให้ RabbitMQ และ Mongo เป็น healthy ก่อน) |
| `rabbitmq` ออก (exit) ทันทีหลังเริ่มใหม่ (มี `.erlang.cookie: eacces` ใน `docker compose logs rabbitmq`) | เป็นปัญหาที่เกิดเป็นครั้งคราวตอนเริ่มครั้งแรก รัน `docker compose up -d` ซ้ำอีกครั้ง |
| อยากเริ่มใหม่ทั้งหมด | `make clean && make up` |
