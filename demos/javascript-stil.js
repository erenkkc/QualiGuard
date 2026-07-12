/**
 * Sunum demosu — çok sayıda var uyarısı, kapı GEÇER.
 * (Gerçek ödev dosyası için: testdata/mehmet_script.js — ~46 uyarı)
 */
document.addEventListener('DOMContentLoaded', function () {
  initNav();
  initForm();
});

function initNav() {
  var header = document.querySelector('.header');
  var menu = document.querySelector('.menu');
  if (!header) return;
  window.addEventListener('scroll', function () {
    if (window.scrollY > 50) {
      header.classList.add('scrolled');
    } else {
      header.classList.remove('scrolled');
    }
  });
  if (menu) {
    menu.innerHTML = '<span>Ana Sayfa</span>';
  }
}

function initForm() {
  var form = document.querySelector('form');
  var name = document.getElementById('name');
  if (form) {
    form.addEventListener('submit', function (e) {
      e.preventDefault();
      if (name.value == '') {
        alert('İsim gerekli');
      }
    });
  }
}
