const form = document.getElementById("userRegisterForm");
const msg = document.getElementById("userRegisterMsg");

if (form && msg) {
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const nickname = document.getElementById("nickname").value.trim();
    const userId = document.getElementById("userId").value.trim();
    const password = document.getElementById("password").value;
    msg.textContent = "\u6b63\u5728\u63d0\u4ea4\u6ce8\u518c\u7533\u8bf7...";
    try {
      const res = await fetch("/api/users/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ nickname, user_id: userId, password }),
        credentials: "same-origin",
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data.detail || "注册失败");
      msg.textContent = data.message || "\u6ce8\u518c\u7533\u8bf7\u5df2\u63d0\u4ea4\uff0c\u8bf7\u7b49\u5f85\u7ba1\u7406\u5458\u5ba1\u6838\u901a\u8fc7\u540e\u518d\u767b\u5f55\u3002";
      form.reset();
    } catch (err) {
      msg.textContent = err.message || "注册失败";
    }
  });
}
